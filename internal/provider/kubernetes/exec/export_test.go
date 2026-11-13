// Copyright 2019 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package exec

import (
	"context"
	"io"
	"os"

	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/remotecommand"
)

var (
	ProcessEnv                   = processEnv
	NewForTest                   = newClient
	HandleContainerNotFoundError = handleContainerNotFoundError
	RandomString                 = &randomString
)

func (ep *ExecParams) Validate(ctx context.Context, podGetter typedcorev1.PodInterface) error {
	return ep.validate(ctx, podGetter)
}

func (fr *FileResource) Validate() error {
	return fr.validate()
}

func (cp *CopyParams) Validate() error {
	return cp.validate()
}

func SafeRun(ctx context.Context, opts ExecParams, executor remotecommand.Executor) error {
	return client{}.safeRun(ctx, opts, opts.Stdin, executor)
}

type SizeQueueInterface interface {
	Next() *remotecommand.TerminalSize
	Push(remotecommand.TerminalSize) bool
	Watch(int)
	Stop()
}

func (s *sizeQueue) Push(size remotecommand.TerminalSize) bool {
	return s.push(size)
}

func (s *sizeQueue) Watch(fd int) {
	s.watch(fd)
}

func (s *sizeQueue) Stop() {
	s.stop()
}

func NewSizeQueueForTest(resizeChan chan remotecommand.TerminalSize, getSize SizeGetter, nCh chan os.Signal) SizeQueueInterface {
	return &sizeQueue{
		resizeChan: resizeChan,
		done:       make(chan struct{}),
		getSize:    getSize,
		nCh:        nCh,
	}
}

const MaxStdinReplay = maxStdinReplay

type AttemptStdin = attemptStdin

var NewAttemptStdin = newAttemptStdin

func (s *attemptStdin) Next() io.Reader {
	return s.next()
}

func (s *attemptStdin) Retry() (io.Reader, bool) {
	return s.retry()
}

var NewAttemptSizeQueue = newAttemptSizeQueue

func (s *attemptSizeQueue) NextAttempt() TerminalSizeQueue {
	return s.next()
}

func (s *attemptSizeQueue) Stop() {
	s.stop()
}
