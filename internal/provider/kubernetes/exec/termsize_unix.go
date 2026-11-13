// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

//go:build !windows

package exec

import (
	"context"
	"os"
	"os/signal"

	"github.com/juju/errors"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/unix"
	"k8s.io/client-go/tools/remotecommand"
)

func getTermSize(fd int) (*remotecommand.TerminalSize, error) {
	w, h, err := terminal.GetSize(fd)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &remotecommand.TerminalSize{Width: uint16(w), Height: uint16(h)}, nil
}

type sizeQueue struct {
	getSize    SizeGetter
	nCh        chan os.Signal
	done       chan struct{}
	resizeChan chan remotecommand.TerminalSize
}

type getSize struct{}

func (g *getSize) Get(fd int) *remotecommand.TerminalSize {
	size, err := getTermSize(fd)
	if err != nil {
		// Ignores error and return nil size.
		logger.Debugf(context.TODO(), "unable to get terminal size: %v", err)
	}
	return size
}

func newSizeQueue() sizeQueueInterface {
	return &sizeQueue{
		resizeChan: make(chan remotecommand.TerminalSize, 1),
		done:       make(chan struct{}),
		getSize:    &getSize{},
		nCh:        make(chan os.Signal, 1),
	}
}

var _ remotecommand.TerminalSizeQueue = (*sizeQueue)(nil)

// Next returns the next terminal size, or nil once monitoring stops.
func (s *sizeQueue) Next() *remotecommand.TerminalSize {
	// Prefer a completed stop over an already queued size.
	select {
	case <-s.done:
		return nil
	default:
	}
	select {
	case <-s.done:
		return nil
	case size, ok := <-s.resizeChan:
		if !ok {
			return nil
		}
		return &size
	}
}

func (s *sizeQueue) stop() {
	close(s.done)
}

func (s *sizeQueue) watch(fd int) {
	if size := s.getSize.Get(fd); size != nil {
		// Push initial size.
		if !s.push(*size) {
			return
		}
	}

	go func() {
		signal.Notify(s.nCh, unix.SIGWINCH)
		defer signal.Stop(s.nCh)

		for {
			select {
			case <-s.nCh:
				size := s.getSize.Get(fd)
				if size == nil {
					return
				}
				if !s.push(*size) {
					return
				}
			case <-s.done:
				return
			}
		}
	}()
}

func (s *sizeQueue) push(size remotecommand.TerminalSize) bool {
	select {
	case <-s.done:
		return false
	default:
	}
	select {
	case <-s.done:
		return false
	case s.resizeChan <- size:
		return true
	}
}
