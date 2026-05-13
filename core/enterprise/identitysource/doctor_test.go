//go:build enterprise

package identitysource

import (
	"context"
	"errors"
	"testing"

	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/stretchr/testify/require"
)

func TestDoctorFailsWhenRequiredCredentialsAreMissing(t *testing.T) {
	result := RunDoctor(context.Background(), EffectiveConfig{
		Provider: enterprisemodels.ProviderFeishu,
		Source:   SourceEnv,
	}, fakeProbe{})

	require.Equal(t, enterprisemodels.IdentitySourceStatusFailed, result.Status)
	requireCheck(t, result, "credentials", CheckFailed)
}

func TestDoctorWarnsWhenUsingEnvSourceForCompatibility(t *testing.T) {
	result := RunDoctor(context.Background(), EffectiveConfig{
		Provider:    enterprisemodels.ProviderFeishu,
		Source:      SourceEnv,
		AppID:       "app",
		AppSecret:   "secret",
		RedirectURI: "https://example.com/callback",
		FrontendURL: "https://example.com",
	}, fakeProbe{tenant: TenantProbeResult{TenantKey: "tenant-a", Name: "Tenant A"}})

	require.Equal(t, enterprisemodels.IdentitySourceStatusWarning, result.Status)
	requireCheck(t, result, "source", CheckWarning)
	requireCheck(t, result, "tenant", CheckPassed)
	requireCheck(t, result, "departments", CheckPassed)
	requireCheck(t, result, "users", CheckPassed)
}

func TestDoctorWarnsOnTenantMismatch(t *testing.T) {
	result := RunDoctor(context.Background(), EffectiveConfig{
		Provider:      enterprisemodels.ProviderFeishu,
		Source:        SourceDB,
		ExternalOrgID: "expected-tenant",
		AppID:         "app",
		AppSecret:     "secret",
		RedirectURI:   "https://example.com/callback",
		FrontendURL:   "https://example.com",
	}, fakeProbe{tenant: TenantProbeResult{TenantKey: "actual-tenant", Name: "Tenant A"}})

	require.Equal(t, enterprisemodels.IdentitySourceStatusWarning, result.Status)
	requireCheck(t, result, "tenant_match", CheckWarning)
}

func TestDoctorReportsPermissionProbeFailure(t *testing.T) {
	result := RunDoctor(context.Background(), EffectiveConfig{
		Provider:    enterprisemodels.ProviderFeishu,
		Source:      SourceDB,
		AppID:       "app",
		AppSecret:   "secret",
		RedirectURI: "https://example.com/callback",
		FrontendURL: "https://example.com",
	}, fakeProbe{departmentsErr: errors.New("missing department permission")})

	require.Equal(t, enterprisemodels.IdentitySourceStatusFailed, result.Status)
	requireCheck(t, result, "departments", CheckFailed)
}

type fakeProbe struct {
	tenant         TenantProbeResult
	tenantErr      error
	departmentsErr error
	usersErr       error
}

func (p fakeProbe) Tenant(ctx context.Context, cfg EffectiveConfig) (TenantProbeResult, error) {
	return p.tenant, p.tenantErr
}

func (p fakeProbe) Departments(ctx context.Context, cfg EffectiveConfig) error {
	return p.departmentsErr
}

func (p fakeProbe) Users(ctx context.Context, cfg EffectiveConfig) error {
	return p.usersErr
}

func requireCheck(t *testing.T, result DoctorResult, code string, level string) {
	t.Helper()

	for _, item := range result.Checks {
		if item.Code == code {
			require.Equal(t, level, item.Level)
			return
		}
	}

	t.Fatalf("check %q not found in %#v", code, result.Checks)
}
