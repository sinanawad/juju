// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// Package advisor implements the `juju advisor` operator CLI command.
//
// The advisor command surfaces deployment-level findings:
// degradations caused by external factors (charms, infrastructure) that
// are invisible to `juju status`. A Finding is a structured record with
// a stable schema (severity, owner, entity, summary, recommendation,
// protocol_ref, plus check_id and entity_kind) emitted by a Detector
// running over the current `params.FullStatus` snapshot. Three v1
// detectors ship: active-with-message, charm-revision-aging, and
// unit-blocked-stale.
//
// See github.com/juju/juju/cmd/juju/status for the related `juju
// status` command this complements. See the sections below for the
// detection-layer flow that spans Finding, Detector, formatter, and
// optional fixture enrichment.
//
// # Detection flow
//
//	user types -> Run() -> Client.Status -> detectors run -> Enrich
//	            -> severity filter -> sort -> formatter -> stdout
//
// Detectors are pure functions over `*params.FullStatus` and an
// injected `time.Time`. They never mutate the status response.
// Enrichment is a post-detection transform on the Finding's
// `recommendation` field; the layer is optional and bypassed by
// `--no-ai`. The CLI does not care whether enrichment ran.
package advisor
