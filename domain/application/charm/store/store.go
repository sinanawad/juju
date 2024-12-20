// Copyright 2024 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package store

import (
	"context"
	"encoding/base64"
	"io"
	"os"

	"github.com/juju/juju/core/objectstore"
	applicationerrors "github.com/juju/juju/domain/application/errors"
	"github.com/juju/juju/internal/errors"
	objectstoreerrors "github.com/juju/juju/internal/objectstore/errors"
	"github.com/juju/juju/internal/uuid"
)

const (
	// ErrNotFound is returned when the file is not found.
	ErrNotFound = errors.ConstError("file not found")
)

// CharmStore provides an API for storing and retrieving charm blobs.
type CharmStore struct {
	objectStoreGetter objectstore.ModelObjectStoreGetter
	encoder           *base64.Encoding
}

// NewCharmStore returns a new charm store instance.
func NewCharmStore(objectStoreGetter objectstore.ModelObjectStoreGetter) *CharmStore {
	return &CharmStore{
		objectStoreGetter: objectStoreGetter,
		encoder:           base64.StdEncoding.WithPadding(base64.NoPadding),
	}
}

// Store the charm at the specified path into the object store. It is expected
// that the archive already exists at the specified path. If the file isn't
// found, a [ErrNotFound] is returned.
func (s *CharmStore) Store(ctx context.Context, path string, size int64, hash string) (string, objectstore.UUID, error) {
	objectStore, err := s.objectStoreGetter.GetObjectStore(ctx)
	if err != nil {
		return "", "", errors.Errorf("getting object store: %w", err)
	}

	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", "", errors.Errorf("%q: %w", path, ErrNotFound)
	} else if err != nil {
		return "", "", errors.Errorf("opening file %q: %w", path, err)
	}

	// Ensure that we close any open handles to the file.
	defer file.Close()

	// Generate a unique path for the file.
	unique, err := uuid.NewUUID()
	if err != nil {
		return "", "", errors.Errorf("cannot generate unique path")
	}
	uniqueName := s.encoder.EncodeToString(unique[:])

	// Store the file in the object store.
	uuid, err := objectStore.PutAndCheckHash(ctx, uniqueName, file, size, hash)
	if err != nil {
		return "", "", errors.Errorf("putting charm: %w", err)
	}
	return uniqueName, uuid, nil
}

// Get retrieves a ReadCloser for the charm archive at the give path from
// the underlying storage.
// NOTE: It is up to the caller to verify the integrity of the data from the charm
// hash stored in DQLite.
func (s *CharmStore) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	store, err := s.objectStoreGetter.GetObjectStore(ctx)
	if err != nil {
		return nil, errors.Errorf("getting object store: %w", err)
	}
	reader, _, err := store.Get(ctx, path)
	if errors.Is(err, objectstoreerrors.ObjectNotFound) {
		return nil, applicationerrors.CharmNotFound
	}
	if err != nil {
		return nil, errors.Errorf("getting charm: %w", err)
	}
	return reader, nil
}

// GetBySHA256Prefix retrieves a ReadCloser for a charm archive who's SHA256 hash
// starts with the provided prefix.
func (s *CharmStore) GetBySHA256Prefix(ctx context.Context, sha256Prefix string) (io.ReadCloser, error) {
	store, err := s.objectStoreGetter.GetObjectStore(ctx)
	if err != nil {
		return nil, errors.Errorf("getting object store: %w", err)
	}
	reader, _, err := store.GetBySHA256Prefix(ctx, sha256Prefix)
	if errors.Is(err, objectstoreerrors.ObjectNotFound) {
		return nil, applicationerrors.CharmNotFound
	}
	if err != nil {
		return nil, errors.Errorf("getting charm: %w", err)
	}
	return reader, nil
}
