// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package exec_test

import (
	"context"
	"testing"

	"github.com/juju/tc"

	"github.com/juju/juju/internal/provider/kubernetes/exec"
	coretesting "github.com/juju/juju/internal/testing"
)

type resizeSuite struct {
	coretesting.BaseSuite
}

func TestResizeSuite(t *testing.T) {
	tc.Run(t, &resizeSuite{})
}

func (s *resizeSuite) TestNoQueue(c *tc.C) {
	queue := exec.NewAttemptSizeQueue(nil)
	c.Check(queue == nil, tc.IsTrue)
	c.Check(queue.NextAttempt(), tc.IsNil)
	queue.Stop()
}

func (s *resizeSuite) TestRetryReplaysLatestSize(c *tc.C) {
	source := newControlledSizeQueue(c.Context())
	queue := exec.NewAttemptSizeQueue(source)
	defer queue.Stop()
	first := queue.NextAttempt()
	initial := exec.TerminalSize{Width: 80, Height: 24}
	latest := exec.TerminalSize{Width: 120, Height: 40}

	for _, size := range []exec.TerminalSize{initial, latest} {
		got := readSize(first)
		<-source.entered
		source.sizes <- &size
		c.Check(<-got, tc.DeepEquals, &size)
	}

	second := queue.NextAttempt()
	c.Check(first.Next(), tc.IsNil)
	c.Check(second.Next(), tc.DeepEquals, &latest)

	next := exec.TerminalSize{Width: 160, Height: 50}
	got := readSize(second)
	<-source.entered
	source.sizes <- &next
	c.Check(<-got, tc.DeepEquals, &next)
}

func (s *resizeSuite) TestRetryReceivesPendingInitialSize(c *tc.C) {
	source := newControlledSizeQueue(c.Context())
	queue := exec.NewAttemptSizeQueue(source)
	defer queue.Stop()
	first := queue.NextAttempt()
	firstResult := readSize(first)
	// Ensure the failed attempt is already inside the source's Next.
	<-source.entered

	second := queue.NextAttempt()
	secondResult := readSize(second)
	initial := exec.TerminalSize{Width: 80, Height: 24}
	source.sizes <- &initial
	c.Check(<-firstResult, tc.IsNil)
	c.Check(<-secondResult, tc.DeepEquals, &initial)
	c.Check(first.Next(), tc.IsNil)
}

func (s *resizeSuite) TestRetryReceivesPendingResize(c *tc.C) {
	source := newControlledSizeQueue(c.Context())
	queue := exec.NewAttemptSizeQueue(source)
	defer queue.Stop()
	first := queue.NextAttempt()
	initial := exec.TerminalSize{Width: 80, Height: 24}
	firstResult := readSize(first)
	<-source.entered
	source.sizes <- &initial
	c.Check(<-firstResult, tc.DeepEquals, &initial)

	// The old stream is waiting for a resize when its exec fails.
	firstResult = readSize(first)
	<-source.entered
	second := queue.NextAttempt()
	c.Check(second.Next(), tc.DeepEquals, &initial)
	secondResult := readSize(second)
	latest := exec.TerminalSize{Width: 120, Height: 40}
	source.sizes <- &latest
	c.Check(<-firstResult, tc.IsNil)
	c.Check(<-secondResult, tc.DeepEquals, &latest)
}

func (s *resizeSuite) TestMultipleRetriesDetachWaitingViews(c *tc.C) {
	source := newControlledSizeQueue(c.Context())
	queue := exec.NewAttemptSizeQueue(source)
	defer queue.Stop()
	firstResult := readSize(queue.NextAttempt())
	<-source.entered
	secondResult := readSize(queue.NextAttempt())
	thirdResult := readSize(queue.NextAttempt())
	c.Check(<-secondResult, tc.IsNil)

	latest := exec.TerminalSize{Width: 120, Height: 40}
	source.sizes <- &latest
	c.Check(<-firstResult, tc.IsNil)
	c.Check(<-thirdResult, tc.DeepEquals, &latest)
}

func (s *resizeSuite) TestCompletedSourceReplaysLatestSize(c *tc.C) {
	source := newControlledSizeQueue(c.Context())
	queue := exec.NewAttemptSizeQueue(source)
	defer queue.Stop()
	first := queue.NextAttempt()
	initial := exec.TerminalSize{Width: 80, Height: 24}
	got := readSize(first)
	<-source.entered
	source.sizes <- &initial
	c.Check(<-got, tc.DeepEquals, &initial)

	got = readSize(first)
	<-source.entered
	source.sizes <- nil
	c.Check(<-got, tc.IsNil)
	c.Check(first.Next(), tc.IsNil)

	second := queue.NextAttempt()
	c.Check(second.Next(), tc.DeepEquals, &initial)
	c.Check(second.Next(), tc.IsNil)
}

func (s *resizeSuite) TestPendingSourceCompletionStopsRetry(c *tc.C) {
	source := newControlledSizeQueue(c.Context())
	queue := exec.NewAttemptSizeQueue(source)
	defer queue.Stop()
	firstResult := readSize(queue.NextAttempt())
	<-source.entered
	second := queue.NextAttempt()
	secondResult := readSize(second)
	source.sizes <- nil
	c.Check(<-firstResult, tc.IsNil)
	c.Check(<-secondResult, tc.IsNil)
	c.Check(second.Next(), tc.IsNil)
}

func (s *resizeSuite) TestStopWakesWaitingView(c *tc.C) {
	source := newControlledSizeQueue(c.Context())
	queue := exec.NewAttemptSizeQueue(source)
	firstResult := readSize(queue.NextAttempt())
	<-source.entered
	second := queue.NextAttempt()
	secondResult := readSize(second)

	queue.Stop()
	c.Check(<-secondResult, tc.IsNil)
	c.Check(second.Next(), tc.IsNil)
	c.Check(queue.NextAttempt().Next(), tc.IsNil)
	queue.Stop()

	// The source's owner releases its in-flight read when the session ends.
	source.sizes <- &exec.TerminalSize{Width: 80, Height: 24}
	c.Check(<-firstResult, tc.IsNil)
}

func (s *resizeSuite) TestSourceCancellationStopsAllViews(c *tc.C) {
	ctx, cancel := context.WithCancel(c.Context())
	defer cancel()
	source := newControlledSizeQueue(ctx)
	queue := exec.NewAttemptSizeQueue(source)
	defer queue.Stop()
	firstResult := readSize(queue.NextAttempt())
	<-source.entered
	secondResult := readSize(queue.NextAttempt())
	cancel()
	c.Check(<-firstResult, tc.IsNil)
	c.Check(<-secondResult, tc.IsNil)
}

func (s *resizeSuite) TestRetainedSizeIsCopied(c *tc.C) {
	source := newControlledSizeQueue(c.Context())
	queue := exec.NewAttemptSizeQueue(source)
	defer queue.Stop()
	initial := exec.TerminalSize{Width: 80, Height: 24}
	got := readSize(queue.NextAttempt())
	<-source.entered
	source.sizes <- &initial
	result := <-got
	c.Assert(result, tc.NotNil)
	result.Width = 1
	initial.Width = 2
	c.Check(queue.NextAttempt().Next(), tc.DeepEquals,
		&exec.TerminalSize{Width: 80, Height: 24})
}

// controlledSizeQueue acknowledges entry into Next so tests can detach an
// attempt while it is reading. Its unbuffered channels also catch concurrent
// source reads: a second call would wait on entered instead of delivering a
// size to the active attempt.
type controlledSizeQueue struct {
	ctx     context.Context
	entered chan struct{}
	sizes   chan *exec.TerminalSize
}

func newControlledSizeQueue(ctx context.Context) *controlledSizeQueue {
	return &controlledSizeQueue{
		ctx:     ctx,
		entered: make(chan struct{}),
		sizes:   make(chan *exec.TerminalSize),
	}
}

func (q *controlledSizeQueue) Next() *exec.TerminalSize {
	select {
	case q.entered <- struct{}{}:
	case <-q.ctx.Done():
		return nil
	}
	select {
	case size := <-q.sizes:
		return size
	case <-q.ctx.Done():
		return nil
	}
}
