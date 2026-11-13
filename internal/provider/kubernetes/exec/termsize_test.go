// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

//go:build aix || dragonfly || freebsd || (js && wasm) || linux || netbsd || openbsd || solaris

package exec_test

import (
	"os"
	"testing"

	"github.com/canonical/gomock/gomock"
	"github.com/juju/tc"
	"golang.org/x/sys/unix"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/juju/juju/internal/provider/kubernetes/exec"
	"github.com/juju/juju/internal/provider/kubernetes/exec/mocks"
	coretesting "github.com/juju/juju/internal/testing"
)

type termSizeSuite struct {
	coretesting.BaseSuite

	sizeQueue exec.SizeQueueInterface
	getSize   *mocks.MockSizeGetter
	nCh       chan os.Signal
}

func TestTermSizeSuite(t *testing.T) {
	tc.Run(t, &termSizeSuite{})
}

func (s *termSizeSuite) TearDownTest(c *tc.C) {
	if s.sizeQueue != nil {
		s.sizeQueue.Stop()
		s.sizeQueue = nil
	}
	s.getSize = nil
	s.nCh = nil
	s.BaseSuite.TearDownTest(c)
}

func (s *termSizeSuite) TestWatch(c *tc.C) {
	ctrl := s.setupQ(c)
	defer ctrl.Finish()
	size1 := &remotecommand.TerminalSize{Width: 111, Height: 222}
	size2 := &remotecommand.TerminalSize{Width: 333, Height: 666}
	gomock.InOrder(
		s.getSize.EXPECT().Get(1).Return(size1),
		s.getSize.EXPECT().Get(1).Return(size2),
	)

	s.sizeQueue.Watch(1)
	c.Check(s.sizeQueue.Next(), tc.DeepEquals, size1)
	s.nCh <- unix.SIGWINCH
	c.Check(s.sizeQueue.Next(), tc.DeepEquals, size2)
}

func (s *termSizeSuite) TestStopUnblocksNext(c *tc.C) {
	ctrl := s.setupQ(c)
	defer ctrl.Finish()
	queue := s.sizeQueue
	result := readSize(queue)

	queue.Stop()
	s.sizeQueue = nil
	c.Check(<-result, tc.IsNil)
	c.Check(queue.Next(), tc.IsNil)
}

func (s *termSizeSuite) TestStopDiscardsBufferedSize(c *tc.C) {
	ctrl := s.setupQ(c)
	defer ctrl.Finish()
	queue := s.sizeQueue
	c.Check(queue.Push(remotecommand.TerminalSize{Width: 80, Height: 24}),
		tc.IsTrue)

	queue.Stop()
	s.sizeQueue = nil
	c.Check(queue.Next(), tc.IsNil)
}

func (s *termSizeSuite) TestStopUnblocksPush(c *tc.C) {
	ctrl := s.setupQ(c)
	defer ctrl.Finish()
	queue := s.sizeQueue
	size := remotecommand.TerminalSize{Width: 80, Height: 24}
	c.Check(queue.Push(size), tc.IsTrue)
	// The buffer is full, so this push cannot complete before stop.
	result := make(chan bool, 1)
	go func() {
		result <- queue.Push(size)
	}()

	queue.Stop()
	s.sizeQueue = nil
	c.Check(<-result, tc.IsFalse)
}

func (s *termSizeSuite) TestStoppedQueueRejectsPush(c *tc.C) {
	ctrl := s.setupQ(c)
	defer ctrl.Finish()
	queue := s.sizeQueue
	queue.Stop()
	s.sizeQueue = nil

	// Even with room in the queue, a stopped stream needs no more sizes.
	c.Check(queue.Push(remotecommand.TerminalSize{Width: 80, Height: 24}),
		tc.IsFalse)
}

func (s *termSizeSuite) TestStopUnblocksInitialWatch(c *tc.C) {
	ctrl := s.setupQ(c)
	defer ctrl.Finish()
	// An unbuffered queue makes watch wait while pushing its initial size.
	s.sizeQueue = exec.NewSizeQueueForTest(
		make(chan remotecommand.TerminalSize), s.getSize, s.nCh,
	)
	queue := s.sizeQueue
	gettingSize := make(chan struct{})
	s.getSize.EXPECT().Get(1).DoAndReturn(func(int) *remotecommand.TerminalSize {
		close(gettingSize)
		return &remotecommand.TerminalSize{Width: 80, Height: 24}
	})
	watchReturned := make(chan struct{})
	go func() {
		defer close(watchReturned)
		queue.Watch(1)
	}()
	<-gettingSize

	queue.Stop()
	s.sizeQueue = nil
	<-watchReturned
	c.Check(queue.Next(), tc.IsNil)
}

func (s *termSizeSuite) setupQ(c *tc.C) *gomock.Controller {
	ctrl := gomock.NewController(c)
	s.getSize = mocks.NewMockSizeGetter(ctrl)
	s.nCh = make(chan os.Signal, 1)
	s.sizeQueue = exec.NewSizeQueueForTest(
		make(chan remotecommand.TerminalSize, 1),
		s.getSize, s.nCh,
	)
	return ctrl
}
