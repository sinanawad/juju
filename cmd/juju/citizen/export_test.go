// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package citizen

import (
	"context"
	"time"

	"github.com/juju/clock"

	"github.com/juju/juju/api/jujuclient"
	"github.com/juju/juju/cmd/cmd"
	"github.com/juju/juju/cmd/modelcmd"
)

// NewCitizenCommandForTest returns a wrapped citizen command with the
// given client store, status API, and clock injected. The status API
// is returned by apiFunc, mirroring the block.NewListCommandForTest
// pattern.
func NewCitizenCommandForTest(
	store jujuclient.ClientStore,
	api statusAPI,
	apiErr error,
	ck clock.Clock,
) cmd.Command {
	c := &citizenCommand{
		clock: ck,
	}
	c.apiFunc = func(_ context.Context) (statusAPI, error) {
		return api, apiErr
	}
	c.SetClientStore(store)
	return modelcmd.Wrap(c)
}

// Reexports for testing.
var (
	NoFindingsLiteral    = noFindingsLiteral
	FormatHybrid         = formatHybrid
	FormatTable          = formatTable
	RunDetectors         = runDetectors
	RunStatefulDetectors = runStatefulDetectors
	DetectStatusChurn    = detectStatusChurn
)

// SetTableFormatTestOverrides pins the formatter's package-level
// overrides (now, color, model name) for deterministic golden tests.
// The returned restore func reverts every override to the prior value
// in one call.
func SetTableFormatTestOverrides(now func() time.Time, color bool, model string) (restore func()) {
	prevNow, prevColor, prevModel := nowFunc, colorEnabled, modelNameForTest
	nowFunc = now
	colorEnabled = color
	modelNameForTest = model
	return func() {
		nowFunc = prevNow
		colorEnabled = prevColor
		modelNameForTest = prevModel
	}
}

// StatusHistoryAPI is the test-visible alias of the internal
// statusHistoryAPI interface, so tests can declare fakes that
// satisfy it.
type StatusHistoryAPI = statusHistoryAPI

// LoadFixtureForTest exposes the embedded fixture loader to tests.
func LoadFixtureForTest() (map[string]string, error) {
	return loadFixture()
}
