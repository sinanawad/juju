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
