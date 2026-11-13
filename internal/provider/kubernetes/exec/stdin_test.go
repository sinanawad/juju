// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package exec_test

import (
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/juju/tc"

	"github.com/juju/juju/internal/provider/kubernetes/exec"
	coretesting "github.com/juju/juju/internal/testing"
)

type stdinSuite struct {
	coretesting.BaseSuite
}

func TestStdinSuite(t *testing.T) {
	tc.Run(t, &stdinSuite{})
}

func (s *stdinSuite) TestNoInput(c *tc.C) {
	stdin := exec.NewAttemptStdin(nil)
	c.Check(stdin == nil, tc.IsTrue)
	c.Check(stdin.Next(), tc.IsNil)
	second, ok := stdin.Retry()
	c.Check(ok, tc.IsTrue)
	c.Check(second, tc.IsNil)
}

func (s *stdinSuite) TestSingleAttemptReadsAllInput(c *tc.C) {
	stdin := exec.NewAttemptStdin(strings.NewReader("hello world"))

	got, err := io.ReadAll(stdin.Next())
	c.Assert(err, tc.ErrorIsNil)
	c.Check(string(got), tc.Equals, "hello world")
}

func (s *stdinSuite) TestRetryReplaysConsumedInput(c *tc.C) {
	stdin := exec.NewAttemptStdin(strings.NewReader("hello world"))

	first := stdin.Next()
	buf := make([]byte, 5)
	_, err := io.ReadFull(first, buf)
	c.Assert(err, tc.ErrorIsNil)
	c.Check(string(buf), tc.Equals, "hello")

	second, ok := stdin.Retry()
	c.Assert(ok, tc.IsTrue)
	got, err := io.ReadAll(second)
	c.Assert(err, tc.ErrorIsNil)
	c.Check(string(got), tc.Equals, "hello world")

	// The detached attempt reads nothing more.
	n, err := first.Read(make([]byte, 1))
	c.Check(n, tc.Equals, 0)
	c.Check(err, tc.ErrorIs, io.EOF)
}

func (s *stdinSuite) TestDetachedReadHandsInputToRetry(c *tc.C) {
	for _, historySize := range []int{0, exec.MaxStdinReplay} {
		c.Logf("history size %d", historySize)
		reader, writer := io.Pipe()
		defer reader.Close()
		defer writer.Close()
		source := &notifyingStdinReader{
			Reader:  reader,
			started: make(chan struct{}),
		}
		history := strings.Repeat("x", historySize)
		stdin := exec.NewAttemptStdin(io.MultiReader(
			strings.NewReader(history), source,
		))
		first := stdin.Next()
		_, err := io.CopyN(io.Discard, first, int64(historySize))
		c.Assert(err, tc.ErrorIsNil)

		// Ensure the old attempt is inside the source Read before retrying.
		firstRead := make(chan stdinReadResult, 1)
		go func() {
			n, err := first.Read(make([]byte, 16))
			firstRead <- stdinReadResult{n: n, err: err}
		}()
		<-source.started

		second, ok := stdin.Retry()
		c.Assert(ok, tc.IsTrue)
		secondRead := make(chan string, 1)
		go func() {
			got, err := io.ReadAll(second)
			c.Check(err, tc.ErrorIsNil)
			secondRead <- string(got)
		}()

		// At the replay limit, this late chunk must be handed over instead
		// of overflowing history and disappearing from the retry.
		_, err = writer.Write([]byte("late"))
		c.Assert(err, tc.ErrorIsNil)
		c.Assert(writer.Close(), tc.ErrorIsNil)
		res := <-firstRead
		c.Check(res.n, tc.Equals, 0)
		c.Check(res.err, tc.ErrorIs, io.EOF)
		c.Check(<-secondRead, tc.Equals, history+"late")
	}
}

func (s *stdinSuite) TestInputKeepsFlowingAfterDetach(c *tc.C) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	stdin := exec.NewAttemptStdin(reader)

	first := stdin.Next()
	go func() {
		_, _ = writer.Write([]byte("one"))
	}()
	buf := make([]byte, 3)
	_, err := io.ReadFull(first, buf)
	c.Assert(err, tc.ErrorIsNil)

	second, ok := stdin.Retry()
	c.Assert(ok, tc.IsTrue)
	go func() {
		_, _ = writer.Write([]byte("two"))
		writer.Close()
	}()
	got, err := io.ReadAll(second)
	c.Assert(err, tc.ErrorIsNil)
	c.Check(string(got), tc.Equals, "onetwo")
}

func (s *stdinSuite) TestReadCrossingReplayLimitBeforeRetryIsRefused(c *tc.C) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	source := &notifyingStdinReader{
		Reader:  reader,
		started: make(chan struct{}),
	}
	stdin := exec.NewAttemptStdin(io.MultiReader(
		strings.NewReader(strings.Repeat("x", exec.MaxStdinReplay)), source,
	))
	first := stdin.Next()
	_, err := io.CopyN(io.Discard, first, exec.MaxStdinReplay)
	c.Assert(err, tc.ErrorIsNil)

	firstRead := make(chan stdinReadResult, 1)
	go func() {
		n, err := first.Read(make([]byte, 16))
		firstRead <- stdinReadResult{n: n, err: err}
	}()
	<-source.started
	_, err = writer.Write([]byte("late"))
	c.Assert(err, tc.ErrorIsNil)
	res := <-firstRead
	c.Check(res.n, tc.Equals, 4)
	c.Assert(res.err, tc.ErrorIsNil)

	second, ok := stdin.Retry()
	c.Check(ok, tc.IsFalse)
	c.Check(second, tc.IsNil)
	// Refusing a retry also detaches the failed attempt's input copier.
	n, err := first.Read(make([]byte, 1))
	c.Check(n, tc.Equals, 0)
	c.Check(err, tc.ErrorIs, io.EOF)
}

func (s *stdinSuite) TestLargeInputCannotBeReplayed(c *tc.C) {
	stdin := exec.NewAttemptStdin(
		strings.NewReader(strings.Repeat("x", exec.MaxStdinReplay+1)),
	)

	_, err := io.CopyN(io.Discard, stdin.Next(), exec.MaxStdinReplay+1)
	c.Assert(err, tc.ErrorIsNil)
	second, ok := stdin.Retry()
	c.Check(ok, tc.IsFalse)
	c.Check(second, tc.IsNil)
}

func (s *stdinSuite) TestSourceErrorIsSticky(c *tc.C) {
	for _, data := range []string{"", "input"} {
		for _, readErr := range []error{io.EOF, errors.New("input failed")} {
			c.Logf("input %q, source error %v", data, readErr)
			source := &errorStdinReader{data: data, err: readErr}
			stdin := exec.NewAttemptStdin(source)
			first := stdin.Next()
			buf := make([]byte, 16)
			n, err := first.Read(buf)
			c.Check(string(buf[:n]), tc.Equals, data)
			c.Check(err, tc.ErrorIs, readErr)

			second, ok := stdin.Retry()
			c.Assert(ok, tc.IsTrue)
			if data != "" {
				n, err = second.Read(buf)
				c.Assert(err, tc.ErrorIsNil)
				c.Check(string(buf[:n]), tc.Equals, data)
			}
			for range 2 {
				n, err = second.Read(buf)
				c.Check(n, tc.Equals, 0)
				c.Check(err, tc.ErrorIs, readErr)
			}
			c.Check(source.reads, tc.Equals, 1)
		}
	}
}

type stdinReadResult struct {
	n   int
	err error
}

// notifyingStdinReader signals when the first source read has begun.
type notifyingStdinReader struct {
	io.Reader
	started chan struct{}
	once    sync.Once
}

func (r *notifyingStdinReader) Read(p []byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	return r.Reader.Read(p)
}

// errorStdinReader returns input and an error together in a single read.
type errorStdinReader struct {
	data  string
	err   error
	reads int
}

func (r *errorStdinReader) Read(p []byte) (int, error) {
	r.reads++
	return copy(p, r.data), r.err
}
