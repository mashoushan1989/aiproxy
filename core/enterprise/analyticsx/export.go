//go:build enterprise

package analyticsx

import (
	"context"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

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

func BuildExportWorkbook(dataset ExportDataset, filter Filter) (*excelize.File, error) {
	startTime := time.Unix(filter.StartTimestamp, 0)
	endTime := time.Unix(filter.EndTimestamp, 0)

	workbook := excelize.NewFile()
	headerStyle, _ := workbook.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#E8E8F4"}},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	amountStyle, _ := workbook.NewStyle(&excelize.Style{NumFmt: 4})

	workbook.SetSheetName("Sheet1", "Summary")
	writeExportSummarySheet(workbook, dataset, startTime, endTime, headerStyle)

	if _, err := workbook.NewSheet("Department Details"); err != nil {
		return nil, fmt.Errorf("create department sheet: %w", err)
	}
	writeDepartmentExportSheet(workbook, dataset.DepartmentSummaries, headerStyle, amountStyle)

	if _, err := workbook.NewSheet("User Ranking"); err != nil {
		return nil, fmt.Errorf("create user ranking sheet: %w", err)
	}
	writeUserExportSheet(workbook, dataset.UserRanking, headerStyle, amountStyle)

	if _, err := workbook.NewSheet("Model Breakdown"); err != nil {
		return nil, fmt.Errorf("create model sheet: %w", err)
	}
	writeModelExportSheet(workbook, dataset.ModelDistribution, headerStyle, amountStyle)

	return workbook, nil
}

func writeExportSummarySheet(
	workbook *excelize.File,
	dataset ExportDataset,
	startTime, endTime time.Time,
	headerStyle int,
) {
	writeExportHeader(workbook, "Summary", []string{"Metric", "Value"}, headerStyle)

	var (
		totalRequests int64
		totalTokens   int64
		totalAmount   float64
		activeUsers   int
	)
	for _, item := range dataset.DepartmentSummaries {
		totalRequests += item.RequestCount
		totalTokens += item.TotalTokens
		totalAmount += item.UsedAmount
		activeUsers += item.ActiveUsers
	}

	rows := [][]any{
		{"Report Period", fmt.Sprintf("%s ~ %s", startTime.Format(time.DateOnly), endTime.Format(time.DateOnly))},
		{"Generated At", time.Now().Format(time.DateTime)},
		{"Active Departments", dataset.DepartmentSummaryCount},
		{"Active Users", activeUsers},
		{"Total Requests", totalRequests},
		{"Total Tokens", totalTokens},
		{"Total Amount", fmt.Sprintf("%.4f", totalAmount)},
		{"Export Rows", dataset.TotalRows},
	}
	for rowIndex, row := range rows {
		for colIndex, value := range row {
			workbook.SetCellValue("Summary", exportCellName(colIndex+1, rowIndex+2), value)
		}
	}
	workbook.SetColWidth("Summary", "A", "A", 28)
	workbook.SetColWidth("Summary", "B", "B", 32)
}

func writeDepartmentExportSheet(
	workbook *excelize.File,
	items []DepartmentSummary,
	headerStyle, amountStyle int,
) {
	const sheet = "Department Details"
	writeExportHeader(workbook, sheet, []string{
		"Rank", "Department", "Active Users", "Members", "Requests", "Amount",
		"Total Tokens", "Input Tokens", "Output Tokens", "Success Rate",
		"Avg Cost", "Avg Cost per User",
	}, headerStyle)

	for index, item := range items {
		row := index + 2
		values := []any{
			index + 1,
			item.DepartmentName,
			item.ActiveUsers,
			item.MemberCount,
			item.RequestCount,
			item.UsedAmount,
			item.TotalTokens,
			item.InputTokens,
			item.OutputTokens,
			item.SuccessRate,
			item.AvgCost,
			item.AvgCostPerUser,
		}
		for col, value := range values {
			workbook.SetCellValue(sheet, exportCellName(col+1, row), value)
		}
		workbook.SetCellStyle(sheet, exportCellName(6, row), exportCellName(6, row), amountStyle)
	}
	workbook.SetColWidth(sheet, "A", "A", 6)
	workbook.SetColWidth(sheet, "B", "B", 24)
	workbook.SetColWidth(sheet, "F", "F", 14)
}

func writeUserExportSheet(
	workbook *excelize.File,
	items []UserRankingEntry,
	headerStyle, amountStyle int,
) {
	const sheet = "User Ranking"
	writeExportHeader(workbook, sheet, []string{
		"Rank", "User Name", "Department", "Requests", "Amount", "Total Tokens",
		"Input Tokens", "Output Tokens", "Models Used", "Success Rate",
	}, headerStyle)

	for _, item := range items {
		row := item.Rank + 1
		values := []any{
			item.Rank,
			item.UserName,
			item.DepartmentName,
			item.RequestCount,
			item.UsedAmount,
			item.TotalTokens,
			item.InputTokens,
			item.OutputTokens,
			item.UniqueModels,
			item.SuccessRate,
		}
		for col, value := range values {
			workbook.SetCellValue(sheet, exportCellName(col+1, row), value)
		}
		workbook.SetCellStyle(sheet, exportCellName(5, row), exportCellName(5, row), amountStyle)
	}
	workbook.SetColWidth(sheet, "A", "A", 6)
	workbook.SetColWidth(sheet, "B", "B", 22)
	workbook.SetColWidth(sheet, "C", "C", 22)
}

func writeModelExportSheet(
	workbook *excelize.File,
	items []ModelDistributionEntry,
	headerStyle, amountStyle int,
) {
	const sheet = "Model Breakdown"
	writeExportHeader(workbook, sheet, []string{
		"Model", "Requests", "Amount", "Total Tokens",
		"Input Tokens", "Output Tokens", "Unique Users", "Share",
	}, headerStyle)

	for index, item := range items {
		row := index + 2
		values := []any{
			item.Model,
			item.RequestCount,
			item.UsedAmount,
			item.TotalTokens,
			item.InputTokens,
			item.OutputTokens,
			item.UniqueUsers,
			item.Percentage,
		}
		for col, value := range values {
			workbook.SetCellValue(sheet, exportCellName(col+1, row), value)
		}
		workbook.SetCellStyle(sheet, exportCellName(3, row), exportCellName(3, row), amountStyle)
	}
	workbook.SetColWidth(sheet, "A", "A", 34)
	workbook.SetColWidth(sheet, "C", "C", 14)
}

func writeExportHeader(workbook *excelize.File, sheet string, headers []string, style int) {
	for index, header := range headers {
		cell := exportCellName(index+1, 1)
		workbook.SetCellValue(sheet, cell, header)
		workbook.SetCellStyle(sheet, cell, cell, style)
	}
}

func exportCellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}
