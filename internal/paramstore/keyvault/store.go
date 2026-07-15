// Package keyvault implements paramstore.Store on Azure Key Vault secrets.
// Config values are secrets tagged type=config; secret values type=secret —
// the tag is what maps back onto String/SecureString for listing and masking.
package keyvault

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

// api is the slice of the azsecrets client the store uses — a seam so tests
// fake the wire. *azsecrets.Client satisfies it directly.
type api interface {
	GetSecret(ctx context.Context, name, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error)
	SetSecret(ctx context.Context, name string, parameters azsecrets.SetSecretParameters, options *azsecrets.SetSecretOptions) (azsecrets.SetSecretResponse, error)
	DeleteSecret(ctx context.Context, name string, options *azsecrets.DeleteSecretOptions) (azsecrets.DeleteSecretResponse, error)
	GetDeletedSecret(ctx context.Context, name string, options *azsecrets.GetDeletedSecretOptions) (azsecrets.GetDeletedSecretResponse, error)
	PurgeDeletedSecret(ctx context.Context, name string, options *azsecrets.PurgeDeletedSecretOptions) (azsecrets.PurgeDeletedSecretResponse, error)
	NewListSecretPropertiesPager(options *azsecrets.ListSecretPropertiesOptions) *runtime.Pager[azsecrets.ListSecretPropertiesResponse]
}

// Store is the Key Vault-backed paramstore.Store. The read cache mirrors the
// Node CLI's DataLoader (one read per name per run).
type Store struct {
	client api
	cache  map[string]string

	// Soft-delete completion polling (the JS SDK's beginDeleteSecret poller).
	pollInterval time.Duration
	pollTimeout  time.Duration
}

// New builds a Store for the named vault (URL https://<name>.vault.azure.net)
// on DefaultAzureCredential — the same chain the Node CLI used.
func New(vaultName string) (*Store, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("key vault: credential: %w", err)
	}
	client, err := azsecrets.NewClient("https://"+vaultName+".vault.azure.net", cred, nil)
	if err != nil {
		return nil, fmt.Errorf("key vault: client: %w", err)
	}
	return newWithAPI(client), nil
}

func newWithAPI(client api) *Store {
	return &Store{
		client:       client,
		cache:        map[string]string{},
		pollInterval: 2 * time.Second,
		pollTimeout:  2 * time.Minute,
	}
}

// isNotFound reports whether err is a Key Vault 404 — the ONLY error that
// reads may treat as "missing". The Node CLI swallowed every error into ”,
// which made an expired credential indistinguishable from an empty vault.
func isNotFound(err error) bool {
	var respErr *azcore.ResponseError
	return errors.As(err, &respErr) && respErr.StatusCode == 404
}
