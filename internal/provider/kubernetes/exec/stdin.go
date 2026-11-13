// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package exec

import (
	"io"
	"sync"
)

// maxStdinReplay limits the history retained for a lossless retry.
const maxStdinReplay = 64 * 1024

// attemptStdin isolates retries from client-go readers left after a failed
// stream. Each attempt has its own view; retries replay consumed input and
// receive input from detached readers.
//
// A nil *attemptStdin represents no input.
type attemptStdin struct {
	src io.Reader

	mu       sync.Mutex
	cond     *sync.Cond
	attempt  int
	history  []byte
	overflow bool
	pending  []byte
	reading  bool
	err      error
}

func newAttemptStdin(src io.Reader) *attemptStdin {
	if src == nil {
		return nil
	}
	s := &attemptStdin{src: src}
	s.cond = sync.NewCond(&s.mu)
	return s
}

// next returns the initial reader; subsequent attempts must use retry.
func (s *attemptStdin) next() io.Reader {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempt++
	s.cond.Broadcast()
	return &stdinAttempt{stdin: s, attempt: s.attempt}
}

// retry atomically detaches the previous reader and checks the replay limit,
// so late reads cannot invalidate an accepted retry. The previous reader is
// detached even when retry is refused.
func (s *attemptStdin) retry() (io.Reader, bool) {
	if s == nil {
		return nil, true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempt++
	s.cond.Broadcast()
	if s.overflow {
		return nil, false
	}
	return &stdinAttempt{stdin: s, attempt: s.attempt}, true
}

// stdinAttempt is the view of an attemptStdin used by one exec attempt.
type stdinAttempt struct {
	stdin   *attemptStdin
	attempt int
	// replayed counts history bytes delivered to this attempt.
	replayed int
}

// Read returns input for this attempt, or io.EOF if detached.
func (a *stdinAttempt) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	s := a.stdin
	s.mu.Lock()
	for {
		if s.attempt != a.attempt {
			s.mu.Unlock()
			return 0, io.EOF
		}
		if a.attempt > 1 && a.replayed < len(s.history) {
			n := copy(p, s.history[a.replayed:])
			a.replayed += n
			s.mu.Unlock()
			return n, nil
		}
		if len(s.pending) > 0 {
			n := copy(p, s.pending)
			s.pending = s.pending[n:]
			s.record(a.attempt, p[:n])
			s.mu.Unlock()
			return n, nil
		}
		if s.err != nil {
			err := s.err
			s.mu.Unlock()
			return 0, err
		}
		if !s.reading {
			break
		}
		// Wait for the source reader to hand over its in-flight input.
		s.cond.Wait()
	}
	s.reading = true
	s.mu.Unlock()

	n, err := s.src.Read(p)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.reading = false
	if err != nil {
		s.err = err
	}
	defer s.cond.Broadcast()
	if s.attempt != a.attempt {
		s.pending = append(s.pending, p[:n]...)
		return 0, io.EOF
	}
	s.record(a.attempt, p[:n])
	return n, err
}

func (s *attemptStdin) record(attempt int, data []byte) {
	if attempt != 1 || s.overflow {
		return
	}
	if len(s.history)+len(data) > maxStdinReplay {
		s.overflow = true
		return
	}
	s.history = append(s.history, data...)
}
