//go:build enterprise

package analyticsx

import "context"

type ExportDataset struct {
	DepartmentSummaries    []DepartmentSummary
	UserRanking            []UserRankingEntry
	ModelDistribution      []ModelDistributionEntry
	DepartmentSummaryCount int
	UserRankingCount       int
	ModelDistributionCount int
	TotalRows              int
}

func BuildExportDataset(
	ctx context.Context,
	service Service,
	scope Scope,
	filter Filter,
) (ExportDataset, error) {
	departments, err := service.DepartmentSummaries(ctx, scope, filter)
	if err != nil {
		return ExportDataset{}, err
	}

	ranking, _, err := service.UserRanking(ctx, scope, filter)
	if err != nil {
		return ExportDataset{}, err
	}

	distribution, err := service.ModelDistribution(ctx, scope, filter)
	if err != nil {
		return ExportDataset{}, err
	}

	dataset := ExportDataset{
		DepartmentSummaries:    departments,
		UserRanking:            ranking,
		ModelDistribution:      distribution,
		DepartmentSummaryCount: len(departments),
		UserRankingCount:       len(ranking),
		ModelDistributionCount: len(distribution),
	}
	dataset.TotalRows = dataset.DepartmentSummaryCount +
		dataset.UserRankingCount +
		dataset.ModelDistributionCount

	return dataset, nil
}
