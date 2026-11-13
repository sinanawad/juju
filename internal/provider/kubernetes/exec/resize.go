// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package exec

import "sync"

// attemptSizeQueue prevents client-go readers from failed streams consuming
// a retry's resize events. Each retry first receives the latest size.
//
// No background reader is started. The source owner must unblock Next when
// the session ends. A nil *attemptSizeQueue represents no size queue.
type attemptSizeQueue struct {
	src TerminalSizeQueue

	mu      sync.Mutex
	cond    *sync.Cond
	attempt uint64
	latest  *TerminalSize
	version uint64
	reading bool
	ended   bool
	stopped bool
}

func newAttemptSizeQueue(src TerminalSizeQueue) *attemptSizeQueue {
	if src == nil {
		return nil
	}
	q := &attemptSizeQueue{src: src}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// next detaches the previous view and returns a view for the next attempt.
func (q *attemptSizeQueue) next() TerminalSizeQueue {
	if q == nil {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.attempt++
	q.cond.Broadcast()
	return &sizeQueueAttempt{queue: q, attempt: q.attempt}
}

// stop invalidates views and wakes waiting readers. The source owner must
// unblock any in-flight read; its result will be discarded.
func (q *attemptSizeQueue) stop() {
	if q == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.stopped = true
	q.cond.Broadcast()
}

type sizeQueueAttempt struct {
	queue   *attemptSizeQueue
	attempt uint64
	version uint64
}

// Next returns the latest terminal size, then subsequent updates. Detached
// and stopped views return nil. Only one view reads the source at a time.
func (a *sizeQueueAttempt) Next() *TerminalSize {
	q := a.queue
	q.mu.Lock()
	for {
		if q.stopped || a.attempt != q.attempt {
			q.mu.Unlock()
			return nil
		}
		if a.version < q.version {
			a.version = q.version
			size := *q.latest
			q.mu.Unlock()
			return &size
		}
		if q.ended {
			q.mu.Unlock()
			return nil
		}
		if !q.reading {
			break
		}
		q.cond.Wait()
	}
	q.reading = true
	q.mu.Unlock()

	size := q.src.Next()

	q.mu.Lock()
	defer q.mu.Unlock()
	q.reading = false
	defer q.cond.Broadcast()
	if q.stopped {
		return nil
	}
	if size == nil {
		q.ended = true
		return nil
	}
	// Copy the size to isolate retry state from source and consumer mutations.
	latest := *size
	q.latest = &latest
	q.version++
	if a.attempt != q.attempt {
		return nil
	}
	a.version = q.version
	result := latest
	return &result
}
