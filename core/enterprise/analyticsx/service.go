//go:build enterprise

package analyticsx

import (
	"context"
	"fmt"
	"sort"
	"time"

	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"gorm.io/gorm"
)

type Service struct {
	DB           *gorm.DB
	LogDB        *gorm.DB
	OrgDirectory OrgDirectory
	Timeout      time.Duration
}

type modelSelection struct {
	All bool
	IDs []string
}

func (s Service) DepartmentSummaries(
	ctx context.Context,
	scope Scope,
	filter Filter,
) ([]DepartmentSummary, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	groupIDs, _, err := s.resolveGroupIDs(ctx, scope, filter)
	if err != nil {
		return nil, err
	}
	if len(groupIDs) == 0 {
		return []DepartmentSummary{}, nil
	}

	groups, err := s.groupsByID(ctx, groupIDs, scope.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return []DepartmentSummary{}, nil
	}
	models := effectiveModels(scope, filter)
	if !models.All && len(models.IDs) == 0 {
		return []DepartmentSummary{}, nil
	}

	orgUnits, err := s.orgUnitsByID(ctx, groupOrgUnitIDs(groups))
	if err != nil {
		return nil, err
	}

	type groupAgg struct {
		GroupID      string  `gorm:"column:group_id"`
		RequestCount int64   `gorm:"column:request_count"`
		UsedAmount   float64 `gorm:"column:used_amount"`
		TotalTokens  int64   `gorm:"column:total_tokens"`
		InputTokens  int64   `gorm:"column:input_tokens"`
		OutputTokens int64   `gorm:"column:output_tokens"`
		SuccessCount int64   `gorm:"column:success_count"`
	}

	var results []groupAgg
	query := s.summaryQuery(ctx, filter).
		Select(
			"group_id",
			"SUM(request_count) as request_count",
			"SUM(used_amount) as used_amount",
			"SUM(total_tokens) as total_tokens",
			"SUM(input_tokens) as input_tokens",
			"SUM(output_tokens) as output_tokens",
			"SUM(status2xx_count) as success_count",
		)
	query = query.Where("group_id IN ?", groupIDs)
	if !models.All {
		query = query.Where("model IN ?", models.IDs)
	}
	if err := query.Group("group_id").Find(&results).Error; err != nil {
		return nil, fmt.Errorf("query department summaries: %w", err)
	}

	type groupModel struct {
		GroupID string `gorm:"column:group_id"`
		Model   string `gorm:"column:model"`
	}

	var modelPairs []groupModel
	modelQuery := s.summaryQuery(ctx, filter).Select("DISTINCT group_id, model")
	modelQuery = modelQuery.Where("group_id IN ?", groupIDs)
	if !models.All {
		modelQuery = modelQuery.Where("model IN ?", models.IDs)
	}
	if err := modelQuery.Find(&modelPairs).Error; err != nil {
		return nil, fmt.Errorf("query department models: %w", err)
	}

	deptModels := make(map[string]map[string]struct{})
	for _, pair := range modelPairs {
		group := groups[pair.GroupID]
		if group.OrgUnitID == "" {
			continue
		}
		if deptModels[group.OrgUnitID] == nil {
			deptModels[group.OrgUnitID] = map[string]struct{}{}
		}
		deptModels[group.OrgUnitID][pair.Model] = struct{}{}
	}

	deptAgg := make(map[string]*DepartmentSummary)
	deptActiveGroups := make(map[string]map[string]struct{})
	deptSuccessCount := make(map[string]int64)
	for _, result := range results {
		group := groups[result.GroupID]
		if group.OrgUnitID == "" {
			continue
		}
		summary := deptAgg[group.OrgUnitID]
		if summary == nil {
			orgUnit := orgUnits[group.OrgUnitID]
			summary = &DepartmentSummary{
				DepartmentID:   group.OrgUnitID,
				DepartmentName: group.OrgUnitID,
			}
			if orgUnit != nil {
				summary.DepartmentName = orgUnit.Name
				summary.MemberCount = orgUnit.MemberCount
				summary.Level1DeptID, summary.Level2DeptID = orgLevels(orgUnit)
			}
			deptAgg[group.OrgUnitID] = summary
			deptActiveGroups[group.OrgUnitID] = map[string]struct{}{}
		}

		summary.RequestCount += result.RequestCount
		summary.UsedAmount += result.UsedAmount
		summary.TotalTokens += result.TotalTokens
		summary.InputTokens += result.InputTokens
		summary.OutputTokens += result.OutputTokens
		deptSuccessCount[group.OrgUnitID] += result.SuccessCount
		deptActiveGroups[group.OrgUnitID][result.GroupID] = struct{}{}
	}

	summaries := make([]DepartmentSummary, 0, len(deptAgg))
	for deptID, summary := range deptAgg {
		summary.ActiveUsers = len(deptActiveGroups[deptID])
		summary.UniqueModels = len(deptModels[deptID])
		if summary.RequestCount > 0 {
			summary.SuccessRate = float64(deptSuccessCount[deptID]) / float64(summary.RequestCount) * 100
			summary.AvgCost = summary.UsedAmount / float64(summary.RequestCount)
		}
		if summary.ActiveUsers > 0 {
			summary.AvgCostPerUser = summary.UsedAmount / float64(summary.ActiveUsers)
		}
		summaries = append(summaries, *summary)
	}

	sort.SliceStable(summaries, func(i, j int) bool {
		return summaries[i].DepartmentID < summaries[j].DepartmentID
	})

	return summaries, nil
}

func (s Service) UserRanking(
	ctx context.Context,
	scope Scope,
	filter Filter,
) ([]UserRankingEntry, int, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	groupIDs, _, err := s.resolveGroupIDs(ctx, scope, filter)
	if err != nil {
		return nil, 0, err
	}
	if len(groupIDs) == 0 {
		return []UserRankingEntry{}, 0, nil
	}

	groups, err := s.groupsByID(ctx, groupIDs, scope.WorkspaceID)
	if err != nil {
		return nil, 0, err
	}
	if len(groups) == 0 {
		return []UserRankingEntry{}, 0, nil
	}
	models := effectiveModels(scope, filter)
	if !models.All && len(models.IDs) == 0 {
		return []UserRankingEntry{}, 0, nil
	}

	users, err := s.usersByGroupID(ctx, mapKeys(groups))
	if err != nil {
		return nil, 0, err
	}
	orgUnits, err := s.orgUnitsByID(ctx, groupOrgUnitIDs(groups))
	if err != nil {
		return nil, 0, err
	}

	type groupAgg struct {
		GroupID             string  `gorm:"column:group_id"`
		UsedAmount          float64 `gorm:"column:used_amount"`
		RequestCount        int64   `gorm:"column:request_count"`
		TotalTokens         int64   `gorm:"column:total_tokens"`
		InputTokens         int64   `gorm:"column:input_tokens"`
		OutputTokens        int64   `gorm:"column:output_tokens"`
		CachedTokens        int64   `gorm:"column:cached_tokens"`
		CacheCreationTokens int64   `gorm:"column:cache_creation_tokens"`
		SuccessCount        int64   `gorm:"column:success_count"`
		UniqueModels        int     `gorm:"column:unique_models"`
	}

	var results []groupAgg
	query := s.summaryQuery(ctx, filter).
		Select(
			"group_id",
			"SUM(used_amount) as used_amount",
			"SUM(request_count) as request_count",
			"SUM(total_tokens) as total_tokens",
			"SUM(input_tokens) as input_tokens",
			"SUM(output_tokens) as output_tokens",
			"SUM(cached_tokens) as cached_tokens",
			"SUM(cache_creation_tokens) as cache_creation_tokens",
			"SUM(status2xx_count) as success_count",
			"COUNT(DISTINCT model) as unique_models",
		)
	query = query.Where("group_id IN ?", groupIDs)
	if !models.All {
		query = query.Where("model IN ?", models.IDs)
	}
	if err := query.Group("group_id").Find(&results).Error; err != nil {
		return nil, 0, fmt.Errorf("query user ranking: %w", err)
	}

	usageByGroup := make(map[string]groupAgg, len(results))
	for _, result := range results {
		usageByGroup[result.GroupID] = result
	}

	entries := make([]UserRankingEntry, 0, len(groups))
	for groupID, group := range groups {
		user := users[groupID]
		deptName := ""
		if orgUnit := orgUnits[group.OrgUnitID]; orgUnit != nil {
			deptName = orgUnit.Name
		}
		entry := UserRankingEntry{
			GroupID:        groupID,
			UserName:       group.Name,
			DepartmentID:   group.OrgUnitID,
			DepartmentName: deptName,
		}
		if user != nil && user.Name != "" {
			entry.UserName = user.Name
		}
		if result, ok := usageByGroup[groupID]; ok {
			entry.UsedAmount = result.UsedAmount
			entry.RequestCount = result.RequestCount
			entry.TotalTokens = result.TotalTokens
			entry.InputTokens = result.InputTokens
			entry.OutputTokens = result.OutputTokens
			entry.CachedTokens = result.CachedTokens
			entry.CacheCreationTokens = result.CacheCreationTokens
			entry.UniqueModels = result.UniqueModels
			if result.RequestCount > 0 {
				entry.SuccessRate = float64(result.SuccessCount) / float64(result.RequestCount) * 100
			}
		}
		entries = append(entries, entry)
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].UsedAmount != entries[j].UsedAmount {
			return entries[i].UsedAmount > entries[j].UsedAmount
		}
		if entries[i].UserName != entries[j].UserName {
			return entries[i].UserName < entries[j].UserName
		}
		return entries[i].GroupID < entries[j].GroupID
	})

	total := len(entries)
	if filter.Limit > 0 && len(entries) > filter.Limit {
		entries = entries[:filter.Limit]
	}
	for i := range entries {
		entries[i].Rank = i + 1
	}

	return entries, total, nil
}

func (s Service) ModelDistribution(
	ctx context.Context,
	scope Scope,
	filter Filter,
) ([]ModelDistributionEntry, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	groupIDs, _, err := s.resolveGroupIDs(ctx, scope, filter)
	if err != nil {
		return nil, err
	}
	if len(groupIDs) == 0 {
		return []ModelDistributionEntry{}, nil
	}
	models := effectiveModels(scope, filter)
	if !models.All && len(models.IDs) == 0 {
		return []ModelDistributionEntry{}, nil
	}

	type modelAgg struct {
		Model        string  `gorm:"column:model"`
		RequestCount int64   `gorm:"column:request_count"`
		TotalTokens  int64   `gorm:"column:total_tokens"`
		InputTokens  int64   `gorm:"column:input_tokens"`
		OutputTokens int64   `gorm:"column:output_tokens"`
		UsedAmount   float64 `gorm:"column:used_amount"`
		UniqueUsers  int     `gorm:"column:unique_users"`
	}

	var results []modelAgg
	query := s.summaryQuery(ctx, filter).
		Select(
			"model",
			"SUM(request_count) as request_count",
			"SUM(total_tokens) as total_tokens",
			"SUM(input_tokens) as input_tokens",
			"SUM(output_tokens) as output_tokens",
			"SUM(used_amount) as used_amount",
			"COUNT(DISTINCT group_id) as unique_users",
		)
	query = query.Where("group_id IN ?", groupIDs)
	if !models.All {
		query = query.Where("model IN ?", models.IDs)
	}
	if err := query.Group("model").Order("used_amount DESC").Find(&results).Error; err != nil {
		return nil, fmt.Errorf("query model distribution: %w", err)
	}

	var totalAmount float64
	for _, result := range results {
		totalAmount += result.UsedAmount
	}

	entries := make([]ModelDistributionEntry, 0, len(results))
	for _, result := range results {
		entry := ModelDistributionEntry{
			Model:        result.Model,
			RequestCount: result.RequestCount,
			TotalTokens:  result.TotalTokens,
			InputTokens:  result.InputTokens,
			OutputTokens: result.OutputTokens,
			UsedAmount:   result.UsedAmount,
			UniqueUsers:  result.UniqueUsers,
		}
		if totalAmount > 0 {
			entry.Percentage = result.UsedAmount / totalAmount * 100
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (s Service) resolveGroupIDs(ctx context.Context, scope Scope, filter Filter) ([]string, bool, error) {
	selection := SelectGroupIDs(scope, filter.GroupIDs)
	if !selection.All && len(selection.IDs) == 0 {
		return []string{}, false, nil
	}

	groupIDs := cloneStringSlice(selection.IDs)
	allGroups := selection.All

	if len(filter.OrgUnitIDs) > 0 {
		orgGroupIDs, err := s.orgDirectory().GroupIDsForOrgUnits(
			ctx,
			defaultWorkspaceID(scope.WorkspaceID),
			filter.OrgUnitIDs,
		)
		if err != nil {
			return nil, false, fmt.Errorf("resolve org unit groups: %w", err)
		}
		groupIDs = intersectSelection(groupIDs, allGroups, orgGroupIDs)
		allGroups = false
	}

	if len(filter.UserIDs) > 0 {
		userGroupIDs, err := s.orgDirectory().GroupIDsForUsers(
			ctx,
			defaultWorkspaceID(scope.WorkspaceID),
			filter.UserIDs,
		)
		if err != nil {
			return nil, false, fmt.Errorf("resolve user groups: %w", err)
		}
		groupIDs = intersectSelection(groupIDs, allGroups, userGroupIDs)
		allGroups = false
	}

	if allGroups {
		allWorkspaceGroups, err := s.orgDirectory().AllGroupIDsForWorkspace(ctx, defaultWorkspaceID(scope.WorkspaceID))
		if err != nil {
			return nil, false, fmt.Errorf("resolve workspace groups: %w", err)
		}
		return allWorkspaceGroups, true, nil
	}
	return compactStrings(groupIDs), false, nil
}

func (s Service) summaryQuery(ctx context.Context, filter Filter) *gorm.DB {
	return s.LogDB.WithContext(ctx).
		Model(&model.GroupSummary{}).
		Where("hour_timestamp >= ? AND hour_timestamp <= ?", filter.StartTimestamp, filter.EndTimestamp)
}

func (s Service) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.Timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, s.Timeout)
}

func (s Service) orgDirectory() OrgDirectory {
	if s.OrgDirectory != nil {
		return s.OrgDirectory
	}
	return NewGORMOrgDirectory(s.DB)
}

func (s Service) groupsByID(
	ctx context.Context,
	groupIDs []string,
	workspaceID string,
) (map[string]model.Group, error) {
	var groups []model.Group
	query := s.DB.WithContext(ctx).
		Model(&model.Group{}).
		Where("workspace_id = ? AND status = ?", defaultWorkspaceID(workspaceID), model.GroupStatusEnabled).
		Where("id IN ?", groupIDs)
	if err := query.Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("query groups: %w", err)
	}

	byID := make(map[string]model.Group, len(groups))
	for _, group := range groups {
		byID[group.ID] = group
	}
	return byID, nil
}

func (s Service) orgUnitsByID(
	ctx context.Context,
	orgUnitIDs []string,
) (map[string]*enterprisemodels.OrgUnit, error) {
	orgUnitIDs = compactStrings(orgUnitIDs)
	if len(orgUnitIDs) == 0 {
		return map[string]*enterprisemodels.OrgUnit{}, nil
	}

	var orgUnits []enterprisemodels.OrgUnit
	if err := s.DB.WithContext(ctx).
		Where("id IN ?", orgUnitIDs).
		Find(&orgUnits).Error; err != nil {
		return nil, fmt.Errorf("query org units: %w", err)
	}

	byID := make(map[string]*enterprisemodels.OrgUnit, len(orgUnits))
	for i := range orgUnits {
		byID[orgUnits[i].ID] = &orgUnits[i]
	}
	return byID, nil
}

func (s Service) usersByGroupID(
	ctx context.Context,
	groupIDs []string,
) (map[string]*enterprisemodels.EnterpriseUser, error) {
	if len(groupIDs) == 0 {
		return map[string]*enterprisemodels.EnterpriseUser{}, nil
	}

	var users []enterprisemodels.EnterpriseUser
	if err := s.DB.WithContext(ctx).
		Where("default_group_id IN ?", groupIDs).
		Find(&users).Error; err != nil {
		return nil, fmt.Errorf("query enterprise users: %w", err)
	}

	byGroupID := make(map[string]*enterprisemodels.EnterpriseUser, len(users))
	for i := range users {
		byGroupID[users[i].DefaultGroupID] = &users[i]
	}
	return byGroupID, nil
}

func intersectSelection(current []string, currentAll bool, next []string) []string {
	next = compactStrings(next)
	if currentAll {
		return next
	}

	allowed := make(map[string]struct{}, len(current))
	for _, id := range current {
		allowed[id] = struct{}{}
	}

	out := make([]string, 0, len(next))
	for _, id := range next {
		if _, ok := allowed[id]; ok {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func effectiveModels(scope Scope, filter Filter) modelSelection {
	requested := compactStrings(filter.Models)
	allowed := compactStrings(scope.AllowedModels)
	if len(allowed) == 0 {
		if len(requested) == 0 {
			return modelSelection{All: true}
		}
		return modelSelection{IDs: requested}
	}
	if len(requested) == 0 {
		return modelSelection{IDs: allowed}
	}

	allowedSet := make(map[string]struct{}, len(allowed))
	for _, modelName := range allowed {
		allowedSet[modelName] = struct{}{}
	}

	out := make([]string, 0, len(requested))
	for _, modelName := range requested {
		if _, ok := allowedSet[modelName]; ok {
			out = append(out, modelName)
		}
	}
	sort.Strings(out)
	return modelSelection{IDs: out}
}

func groupOrgUnitIDs(groups map[string]model.Group) []string {
	orgUnitIDs := make([]string, 0, len(groups))
	for _, group := range groups {
		if group.OrgUnitID != "" {
			orgUnitIDs = append(orgUnitIDs, group.OrgUnitID)
		}
	}
	return compactStrings(orgUnitIDs)
}

func mapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func orgLevels(orgUnit *enterprisemodels.OrgUnit) (string, string) {
	if orgUnit == nil || orgUnit.Path == "" {
		return "", ""
	}

	parts := compactStrings(splitPath(orgUnit.Path))
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func splitPath(path string) []string {
	out := []string{}
	start := 0
	for i := range path {
		if path[i] != '/' {
			continue
		}
		if i > start {
			out = append(out, path[start:i])
		}
		start = i + 1
	}
	if start < len(path) {
		out = append(out, path[start:])
	}
	return out
}
