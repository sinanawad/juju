// Copyright 2025 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package service

import (
	"context"

	"github.com/juju/juju/core/unit"
	"github.com/juju/juju/domain/agentpassword"
	"github.com/juju/juju/internal/errors"
)

// MigrationState is the state required for migrating passwords.
type MigrationState interface {
	// GetAllUnitPasswordHashes returns a map of unit names to password hashes.
	GetAllUnitPasswordHashes(context.Context) (agentpassword.UnitPasswordHashes, error)

	// GetUnitUUID returns the UUID of the unit with the given name, returning
	// an error satisfying [passworderrors.UnitNotFound] if the unit does not
	// exist.
	GetUnitUUID(context.Context, unit.Name) (unit.UUID, error)

	// SetUnitPasswordHash sets the password hash for the given unit.
	SetUnitPasswordHash(context.Context, unit.UUID, agentpassword.PasswordHash) error
}

// MigrationService provides the API for migrating passwords.
type MigrationService struct {
	st MigrationState
}

// NewMigrationService returns a new service reference wrapping the input state.
func NewMigrationService(
	st MigrationState,
) *MigrationService {
	return &MigrationService{
		st: st,
	}
}

// GetAllUnitPasswordHashes returns a map of unit names to password hashes.
func (s *MigrationService) GetAllUnitPasswordHashes(ctx context.Context) (agentpassword.UnitPasswordHashes, error) {
	return s.st.GetAllUnitPasswordHashes(ctx)
}

// SetUnitPasswordHash sets the password hash for the given unit.
func (s *MigrationService) SetUnitPasswordHash(ctx context.Context, unitName unit.Name, passwordHash agentpassword.PasswordHash) error {
	if err := unitName.Validate(); err != nil {
		return err
	}

	unitUUID, err := s.st.GetUnitUUID(ctx, unitName)
	if err != nil {
		return errors.Errorf("getting unit UUID: %w", err)
	}

	return s.st.SetUnitPasswordHash(ctx, unitUUID, passwordHash)
}
