// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package citizen

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/juju/juju/cmd/cmd"
)

//go:embed testdata/findings.json
var fixtureBytes []byte

var (
	fixtureOnce sync.Once
	fixtureMap  map[string]string
	fixtureErr  error
)

// loadFixture decodes the embedded JSON fixture once and returns the
// check_id -> enriched recommendation map. A malformed or missing
// fixture returns an error; callers MUST NOT treat the error as
// fatal (per FR-016).
func loadFixture() (map[string]string, error) {
	fixtureOnce.Do(func() {
		if len(fixtureBytes) == 0 {
			fixtureErr = fmt.Errorf("empty fixture")
			return
		}
		fixtureErr = json.Unmarshal(fixtureBytes, &fixtureMap)
	})
	return fixtureMap, fixtureErr
}

// enrich rewrites the Recommendation field of each finding with the
// fixture entry for its CheckID, when present. Findings whose
// CheckID has no fixture entry are left unchanged. If the fixture
// cannot be loaded, all findings are returned unchanged and a
// warning is written to ctx.Stderr (FR-016 graceful fallback).
func enrich(ctx *cmd.Context, in []Finding) []Finding {
	if len(in) == 0 {
		return in
	}
	fixture, err := loadFixture()
	if err != nil {
		if ctx != nil && ctx.Stderr != nil {
			fmt.Fprintf(ctx.Stderr,
				"WARNING citizenship: AI enrichment skipped: %s\n", err)
		}
		return in
	}
	out := make([]Finding, len(in))
	for i, f := range in {
		if alt, ok := fixture[f.CheckID]; ok && alt != "" {
			f.Recommendation = alt
		}
		out[i] = f
	}
	return out
}
