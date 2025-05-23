// Copyright 2025 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package service

import (
	"context"

	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gomock "go.uber.org/mock/gomock"
	gc "gopkg.in/check.v1"
)

type serviceSuite struct {
	testing.IsolationSuite

	state *MockState
}

var _ = gc.Suite(&serviceSuite{})

func (s *serviceSuite) TestGetSequencesForExport(c *gc.C) {
	defer s.setupMocks(c).Finish()

	s.state.EXPECT().GetSequencesForExport(gomock.Any()).Return(map[string]uint64{"foo": 12}, nil)

	seqs, err := s.state.GetSequencesForExport(context.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(seqs, gc.DeepEquals, map[string]uint64{"foo": 12})
}

func (s *serviceSuite) TestImportSequences(c *gc.C) {
	defer s.setupMocks(c).Finish()

	s.state.EXPECT().ImportSequences(gomock.Any(), map[string]uint64{"foo": 12}).Return(nil)

	err := s.state.ImportSequences(context.Background(), map[string]uint64{"foo": 12})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *serviceSuite) TestRemoveAllSequences(c *gc.C) {
	defer s.setupMocks(c).Finish()

	s.state.EXPECT().RemoveAllSequences(gomock.Any()).Return(nil)

	err := s.state.RemoveAllSequences(context.Background())
	c.Assert(err, jc.ErrorIsNil)
}

func (s *serviceSuite) setupMocks(c *gc.C) *gomock.Controller {
	ctrl := gomock.NewController(c)

	s.state = NewMockState(ctrl)

	return ctrl
}
