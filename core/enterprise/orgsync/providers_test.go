//go:build enterprise

package orgsync

import (
	"context"
	"testing"

	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/stretchr/testify/require"
)

type fakeProviderClient struct{}

func (fakeProviderClient) Identity(context.Context) (TenantIdentity, error) {
	return TenantIdentity{}, nil
}

func (fakeProviderClient) ListOrgUnits(context.Context) ([]OrgUnitRecord, error) {
	return nil, nil
}

func (fakeProviderClient) ListUsers(context.Context) ([]UserRecord, error) {
	return nil, nil
}

func TestProviderRegistryRegistersManualProvider(t *testing.T) {
	registry := NewProviderRegistry()
	expected := fakeProviderClient{}

	registry.Register(enterprisemodels.ProviderManual, func(config Config) (ProviderClient, error) {
		require.Equal(t, Config{"workspace_id": "default"}, config)
		return expected, nil
	})

	client, err := registry.New(enterprisemodels.ProviderManual, Config{"workspace_id": "default"})
	require.NoError(t, err)
	require.Equal(t, expected, client)
}

func TestProviderRegistryRejectsUnknownWeComProvider(t *testing.T) {
	registry := NewProviderRegistry()

	_, err := registry.New(enterprisemodels.ProviderWeCom, nil)
	require.EqualError(t, err, "provider not registered: wecom")
}
