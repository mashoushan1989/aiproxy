//go:build enterprise

package quota

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
)

func setupQuotaTestDB(t *testing.T) {
	t.Helper()

	prevDB := model.DB
	prevLogDB := model.LogDB
	prevUsingSQLite := common.UsingSQLite
	prevRedisEnabled := common.RedisEnabled

	testDB, err := model.OpenSQLite(filepath.Join(t.TempDir(), "enterprise-quota.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	model.DB = testDB
	model.LogDB = testDB
	common.UsingSQLite = true
	common.RedisEnabled = false

	t.Cleanup(func() {
		model.DB = prevDB
		model.LogDB = prevLogDB
		common.UsingSQLite = prevUsingSQLite
		common.RedisEnabled = prevRedisEnabled
	})

	if err := testDB.AutoMigrate(
		&model.Group{},
		&model.Token{},
		&models.FeishuUser{},
		&models.FeishuDepartment{},
		&models.QuotaPolicy{},
		&models.UserQuotaPolicy{},
		&models.DepartmentQuotaPolicy{},
		&models.GroupQuotaPolicy{},
	); err != nil {
		t.Fatalf("migrate test tables: %v", err)
	}
}

func createQuotaUserToken(t *testing.T, openID, groupID, departmentID, keySuffix string) string {
	t.Helper()

	if err := model.DB.Create(&model.Group{
		ID:     groupID,
		Status: model.GroupStatusEnabled,
	}).Error; err != nil {
		t.Fatalf("create group %s: %v", groupID, err)
	}

	key := strings.Repeat("a", 47) + keySuffix
	if err := model.DB.Create(&model.Token{
		Key:                    key,
		Name:                   model.EmptyNullString("token-" + keySuffix),
		GroupID:                groupID,
		Status:                 model.TokenStatusEnabled,
		UsedAmount:             15000,
		PeriodQuota:            10000,
		PeriodType:             model.EmptyNullString(model.PeriodTypeMonthly),
		PeriodLastUpdateTime:   time.Now(),
		PeriodLastUpdateAmount: 0,
	}).Error; err != nil {
		t.Fatalf("create token %s: %v", keySuffix, err)
	}

	if err := model.DB.Create(&models.FeishuUser{
		OpenID:       openID,
		Name:         openID,
		GroupID:      groupID,
		DepartmentID: departmentID,
		Status:       1,
	}).Error; err != nil {
		t.Fatalf("create feishu user %s: %v", openID, err)
	}

	return key
}

func tokenPeriodQuota(t *testing.T, key string) float64 {
	t.Helper()

	token, err := model.GetTokenByKey(key)
	if err != nil {
		t.Fatalf("get token by key: %v", err)
	}

	return token.PeriodQuota
}

func TestSyncPolicyToTokenClearsHardQuotaWhenBlockAtTier3False(t *testing.T) {
	setupQuotaTestDB(t)

	key := createQuotaUserToken(t, "open-direct", "group-direct", "", "1")
	if _, err := model.GetAndValidateToken(key); err == nil {
		t.Fatal("expected stale token period quota to reject before policy sync")
	}

	policy := &models.QuotaPolicy{
		Name:         "price controlled",
		PeriodQuota:  10000,
		PeriodType:   models.PeriodTypeMonthly,
		BlockAtTier3: false,
	}

	syncPolicyToToken("open-direct", policy)

	if got := tokenPeriodQuota(t, key); got != 0 {
		t.Fatalf("period quota after sync = %v, want 0", got)
	}

	if _, err := model.GetAndValidateToken(key); err != nil {
		t.Fatalf("expected token auth to pass after clearing stale hard quota: %v", err)
	}
}

func TestSyncPolicyBindingsToTokensRefreshesUpdatedPolicyBindings(t *testing.T) {
	setupQuotaTestDB(t)

	policy := &models.QuotaPolicy{
		Name:         "updated price controlled",
		PeriodQuota:  10000,
		PeriodType:   models.PeriodTypeMonthly,
		BlockAtTier3: false,
	}
	if err := model.DB.Create(policy).Error; err != nil {
		t.Fatalf("create policy: %v", err)
	}

	directKey := createQuotaUserToken(t, "open-user", "group-user", "", "2")
	if err := model.DB.Create(&models.UserQuotaPolicy{
		OpenID:        "open-user",
		QuotaPolicyID: policy.ID,
	}).Error; err != nil {
		t.Fatalf("create user policy binding: %v", err)
	}

	if err := model.DB.Create(&models.FeishuDepartment{
		DepartmentID: "dept-a",
		Name:         "Dept A",
		Status:       1,
	}).Error; err != nil {
		t.Fatalf("create department: %v", err)
	}

	deptKey := createQuotaUserToken(t, "open-dept", "group-dept", "dept-a", "3")
	if err := model.DB.Create(&models.DepartmentQuotaPolicy{
		DepartmentID:  "dept-a",
		QuotaPolicyID: policy.ID,
	}).Error; err != nil {
		t.Fatalf("create department policy binding: %v", err)
	}

	syncPolicyBindingsToTokens(policy.ID, policy)

	for _, key := range []string{directKey, deptKey} {
		if got := tokenPeriodQuota(t, key); got != 0 {
			t.Fatalf("period quota for %s after binding sync = %v, want 0", key, got)
		}

		if _, err := model.GetAndValidateToken(key); err != nil {
			t.Fatalf("expected token %s to pass after binding sync: %v", key, err)
		}
	}
}

func TestGetPolicyForUserSkipsExpiredUserOverride(t *testing.T) {
	setupQuotaTestDB(t)

	createQuotaUserToken(t, "open-expired", "group-expired", "dept-expired", "4")
	if err := model.DB.Create(&models.FeishuDepartment{
		DepartmentID: "dept-expired",
		Name:         "Expired Dept",
		Status:       1,
	}).Error; err != nil {
		t.Fatalf("create department: %v", err)
	}

	deptPolicy := &models.QuotaPolicy{
		Name:         "department fallback",
		PeriodQuota:  20000,
		PeriodType:   models.PeriodTypeMonthly,
		BlockAtTier3: false,
	}
	if err := model.DB.Create(deptPolicy).Error; err != nil {
		t.Fatalf("create department policy: %v", err)
	}

	userPolicy := &models.QuotaPolicy{
		Name:         "expired user override",
		PeriodQuota:  5000,
		PeriodType:   models.PeriodTypeMonthly,
		BlockAtTier3: true,
	}
	if err := model.DB.Create(userPolicy).Error; err != nil {
		t.Fatalf("create user policy: %v", err)
	}

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	if err := model.DB.Create(&models.DepartmentQuotaPolicy{
		DepartmentID:  "dept-expired",
		QuotaPolicyID: deptPolicy.ID,
		EffectiveAt:   &past,
		ExpiresAt:     &future,
	}).Error; err != nil {
		t.Fatalf("create department binding: %v", err)
	}

	if err := model.DB.Create(&models.UserQuotaPolicy{
		OpenID:        "open-expired",
		QuotaPolicyID: userPolicy.ID,
		EffectiveAt:   &past,
		ExpiresAt:     &past,
	}).Error; err != nil {
		t.Fatalf("create expired user binding: %v", err)
	}

	got, err := GetPolicyForUser(context.Background(), "open-expired")
	if err != nil {
		t.Fatalf("GetPolicyForUser: %v", err)
	}
	if got == nil || got.ID != deptPolicy.ID {
		t.Fatalf("effective policy ID = %v, want %d", got, deptPolicy.ID)
	}
}

func TestExpirePolicyBindingsResyncsUserToFallbackPolicy(t *testing.T) {
	setupQuotaTestDB(t)

	key := createQuotaUserToken(t, "open-reset", "group-reset", "dept-reset", "5")
	if err := model.DB.Create(&models.FeishuDepartment{
		DepartmentID: "dept-reset",
		Name:         "Reset Dept",
		Status:       1,
	}).Error; err != nil {
		t.Fatalf("create department: %v", err)
	}

	fallbackPolicy := &models.QuotaPolicy{
		Name:         "fallback price control",
		PeriodQuota:  20000,
		PeriodType:   models.PeriodTypeMonthly,
		BlockAtTier3: false,
	}
	if err := model.DB.Create(fallbackPolicy).Error; err != nil {
		t.Fatalf("create fallback policy: %v", err)
	}

	expiredPolicy := &models.QuotaPolicy{
		Name:         "expired hard limit",
		PeriodQuota:  5000,
		PeriodType:   models.PeriodTypeMonthly,
		BlockAtTier3: true,
	}
	if err := model.DB.Create(expiredPolicy).Error; err != nil {
		t.Fatalf("create expired policy: %v", err)
	}

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	if err := model.DB.Create(&models.DepartmentQuotaPolicy{
		DepartmentID:  "dept-reset",
		QuotaPolicyID: fallbackPolicy.ID,
		EffectiveAt:   &past,
		ExpiresAt:     &future,
	}).Error; err != nil {
		t.Fatalf("create fallback binding: %v", err)
	}

	if err := model.DB.Create(&models.UserQuotaPolicy{
		OpenID:        "open-reset",
		QuotaPolicyID: expiredPolicy.ID,
		EffectiveAt:   &past,
		ExpiresAt:     &past,
	}).Error; err != nil {
		t.Fatalf("create expired binding: %v", err)
	}

	if err := model.DB.Model(&model.Token{}).
		Where("key = ?", key).
		Updates(map[string]any{
			"period_quota":  float64(5000),
			"period_type":   model.PeriodTypeMonthly,
			"used_amount":   float64(10000),
			"request_count": 1,
		}).Error; err != nil {
		t.Fatalf("seed stale token hard quota: %v", err)
	}

	expirePolicyBindingsOnce(now)

	if got := tokenPeriodQuota(t, key); got != 0 {
		t.Fatalf("period quota after expiry reset = %v, want 0", got)
	}

	var count int64
	if err := model.DB.Model(&models.UserQuotaPolicy{}).
		Where("open_id = ?", "open-reset").
		Count(&count).Error; err != nil {
		t.Fatalf("count user bindings: %v", err)
	}
	if count != 0 {
		t.Fatalf("active user bindings = %d, want 0", count)
	}
}
