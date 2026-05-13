//go:build enterprise

package identitysource

import (
	"context"
	"fmt"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"

	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
)

const (
	CheckPassed  = "passed"
	CheckWarning = "warning"
	CheckFailed  = "failed"
)

type CheckItem struct {
	Code    string `json:"code"`
	Level   string `json:"level"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

type DoctorResult struct {
	Provider      string      `json:"provider"`
	Source        string      `json:"source"`
	Status        string      `json:"status"`
	TenantKey     string      `json:"tenant_key,omitempty"`
	TenantName    string      `json:"tenant_name,omitempty"`
	Checks        []CheckItem `json:"checks"`
	EffectiveNote string      `json:"effective_note,omitempty"`
}

type TenantProbeResult struct {
	TenantKey string
	Name      string
}

type Probe interface {
	Tenant(ctx context.Context, cfg EffectiveConfig) (TenantProbeResult, error)
	Departments(ctx context.Context, cfg EffectiveConfig) error
	Users(ctx context.Context, cfg EffectiveConfig) error
}

func RunDoctor(ctx context.Context, cfg EffectiveConfig, probe Probe) DoctorResult {
	result := DoctorResult{
		Provider: cfg.Provider,
		Source:   cfg.Source,
		Status:   enterprisemodels.IdentitySourceStatusPassed,
	}

	if cfg.Source == SourceEnv {
		result.add(CheckItem{
			Code:    "source",
			Level:   CheckWarning,
			Message: "Using environment configuration for compatibility.",
		})
	}

	if cfg.AppID == "" || cfg.AppSecret == "" {
		result.add(CheckItem{
			Code:    "credentials",
			Level:   CheckFailed,
			Message: "App ID and app secret are required.",
		})
		return result.finalize()
	}

	result.add(CheckItem{Code: "credentials", Level: CheckPassed, Message: "App credentials are configured."})

	if cfg.RedirectURI == "" {
		result.add(CheckItem{Code: "redirect_uri", Level: CheckFailed, Message: "OAuth redirect URI is required."})
	} else {
		result.add(CheckItem{Code: "redirect_uri", Level: CheckPassed, Message: "OAuth redirect URI is configured."})
	}

	if cfg.FrontendURL == "" {
		result.add(CheckItem{Code: "frontend_url", Level: CheckWarning, Message: "Frontend URL is empty; login redirects may be incomplete."})
	} else {
		result.add(CheckItem{Code: "frontend_url", Level: CheckPassed, Message: "Frontend URL is configured."})
	}

	if probe == nil {
		probe = FeishuProbe{}
	}

	tenant, err := probe.Tenant(ctx, cfg)
	if err != nil {
		result.add(CheckItem{Code: "tenant", Level: CheckFailed, Message: "Failed to query tenant information.", Detail: err.Error()})
	} else {
		result.TenantKey = tenant.TenantKey
		result.TenantName = tenant.Name
		result.add(CheckItem{Code: "tenant", Level: CheckPassed, Message: "Tenant information can be queried."})
		if cfg.ExternalOrgID != "" && tenant.TenantKey != "" && cfg.ExternalOrgID != tenant.TenantKey {
			result.add(CheckItem{
				Code:    "tenant_match",
				Level:   CheckWarning,
				Message: "Configured external org ID does not match the app tenant key.",
				Detail:  fmt.Sprintf("configured=%s actual=%s", cfg.ExternalOrgID, tenant.TenantKey),
			})
		} else if cfg.ExternalOrgID != "" {
			result.add(CheckItem{Code: "tenant_match", Level: CheckPassed, Message: "External org ID matches the app tenant key."})
		}
	}

	if err := probe.Departments(ctx, cfg); err != nil {
		result.add(CheckItem{Code: "departments", Level: CheckFailed, Message: "Failed to read departments.", Detail: err.Error()})
	} else {
		result.add(CheckItem{Code: "departments", Level: CheckPassed, Message: "Department read permission is available."})
	}

	if err := probe.Users(ctx, cfg); err != nil {
		result.add(CheckItem{Code: "users", Level: CheckFailed, Message: "Failed to read department users.", Detail: err.Error()})
	} else {
		result.add(CheckItem{Code: "users", Level: CheckPassed, Message: "Department user read permission is available."})
	}

	return result.finalize()
}

func (r *DoctorResult) add(item CheckItem) {
	r.Checks = append(r.Checks, item)
}

func (r DoctorResult) finalize() DoctorResult {
	status := enterprisemodels.IdentitySourceStatusPassed
	for _, item := range r.Checks {
		if item.Level == CheckFailed {
			status = enterprisemodels.IdentitySourceStatusFailed
			break
		}
		if item.Level == CheckWarning && status != enterprisemodels.IdentitySourceStatusFailed {
			status = enterprisemodels.IdentitySourceStatusWarning
		}
	}
	r.Status = status

	return r
}

type FeishuProbe struct{}

func (FeishuProbe) Tenant(ctx context.Context, cfg EffectiveConfig) (TenantProbeResult, error) {
	client := lark.NewClient(cfg.AppID, cfg.AppSecret)
	resp, err := client.Tenant.Tenant.Query(ctx)
	if err != nil {
		return TenantProbeResult{}, err
	}
	if !resp.Success() {
		return TenantProbeResult{}, resp.CodeError
	}

	result := TenantProbeResult{}
	if resp.Data != nil && resp.Data.Tenant != nil {
		if resp.Data.Tenant.TenantKey != nil {
			result.TenantKey = *resp.Data.Tenant.TenantKey
		}
		if resp.Data.Tenant.Name != nil {
			result.Name = *resp.Data.Tenant.Name
		}
	}

	return result, nil
}

func (FeishuProbe) Departments(ctx context.Context, cfg EffectiveConfig) error {
	client := lark.NewClient(cfg.AppID, cfg.AppSecret)
	resp, err := client.Contact.Department.Children(ctx, larkcontact.NewChildrenDepartmentReqBuilder().
		DepartmentIdType("department_id").
		DepartmentId("0").
		PageSize(1).
		Build())
	if err != nil {
		return err
	}
	if !resp.Success() {
		return resp.CodeError
	}

	return nil
}

func (FeishuProbe) Users(ctx context.Context, cfg EffectiveConfig) error {
	client := lark.NewClient(cfg.AppID, cfg.AppSecret)
	resp, err := client.Contact.User.FindByDepartment(ctx, larkcontact.NewFindByDepartmentUserReqBuilder().
		UserIdType("open_id").
		DepartmentIdType("department_id").
		DepartmentId("0").
		PageSize(1).
		Build())
	if err != nil {
		return err
	}
	if !resp.Success() {
		return resp.CodeError
	}

	return nil
}
