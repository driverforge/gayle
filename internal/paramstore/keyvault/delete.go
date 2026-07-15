package keyvault

import (
	"context"
	"fmt"
	"time"

	"github.com/driverforge/gayle/internal/paramstore"
	"github.com/driverforge/gayle/internal/ui"
)

// DeleteParameters deletes each secret and purges its soft-deleted remains.
// The DF-644 asymmetry is deliberate and preserved:
//
//   - A failed DELETE (or an unconfirmed soft-delete) leaves the secret live —
//     that is a hard, per-key error.
//   - A failed PURGE is a warning only: purge can legitimately be forbidden
//     (purge protection, RBAC), and the soft-deleted secret no longer appears
//     in active listings, which is all pruning needs.
func (s *Store) DeleteParameters(ctx context.Context, names []string) error {
	var errs paramstore.KeyErrors
	for _, name := range names {
		kvName := ToKeyVaultName(name)
		if _, err := s.client.DeleteSecret(ctx, kvName, nil); err != nil {
			errs = append(errs, paramstore.KeyError{Key: name, Err: err})
			continue
		}
		if err := s.waitDeleted(ctx, kvName); err != nil {
			errs = append(errs, paramstore.KeyError{Key: name, Err: err})
			continue
		}
		delete(s.cache, name)
		if _, err := s.client.PurgeDeletedSecret(ctx, kvName, nil); err != nil {
			ui.Warn(fmt.Sprintf("Could not purge soft-deleted secret %q: %v", kvName, err))
		}
	}
	return errs.OrNil()
}

// waitDeleted polls until the soft-deleted secret is retrievable — the Go
// equivalent of the JS SDK's beginDeleteSecret poller. Purging before the
// soft-delete completes returns a conflict.
func (s *Store) waitDeleted(ctx context.Context, kvName string) error {
	deadline := time.Now().Add(s.pollTimeout)
	for {
		_, err := s.client.GetDeletedSecret(ctx, kvName, nil)
		if err == nil {
			return nil
		}
		if !isNotFound(err) {
			return fmt.Errorf("waiting for delete of %s: %w", kvName, err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("delete of %s not confirmed after %s", kvName, s.pollTimeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.pollInterval):
		}
	}
}
