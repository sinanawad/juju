// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package store_test

import (
	"testing"

	gc "gopkg.in/check.v1"
)

//go:generate go run go.uber.org/mock/mockgen -typed -package mocks -destination ./mocks/store_mock.go github.com/juju/juju/cmd/juju/application/store CharmAdder,CharmsAPI,DownloadBundleClient,CharmReader
//go:generate go run go.uber.org/mock/mockgen -typed -package mocks -destination ./mocks/charm_mock.go github.com/juju/juju/internal/charm Bundle

func TestPackage(t *testing.T) {
	gc.TestingT(t)
}
