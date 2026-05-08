package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/labring/aiproxy/core/common/consume"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/passthrough"
	"gorm.io/gorm"
)

var (
	startDate    = flag.String("start", "", "Start date (YYYY-MM-DD)")
	endDate      = flag.String("end", "", "End date (YYYY-MM-DD)")
	dryRun       = flag.Bool("dry-run", true, "Dry-run mode (no database updates)")
	batchSize    = flag.Int("batch-size", 1000, "Batch size for processing")
	confirm      = flag.Bool("confirm", false, "Confirm to apply changes")
	outputReport = flag.String("output", "tmp/responses_api_repair.csv", "Output report path")
)

type RepairStats struct {
	TotalLogs            int
	HasDetail            int
	MissingDetail        int
	Processed            int
	Skipped              int
	DeltaCachedAmount    float64
	DeltaReasoningAmount float64
	DeltaUsedAmount      float64
	ModelStats           map[string]*ModelRepairStats
}

type ModelRepairStats struct {
	Model                string
	LogCount             int
	DeltaCachedAmount    float64
	DeltaReasoningAmount float64
	DeltaUsedAmount      float64
}

type RepairRecord struct {
	LogID              int
	RequestID          string
	Model              string
	CreatedAt          time.Time
	OldCachedTokens    int64
	NewCachedTokens    int64
	OldReasoningTokens int64
	NewReasoningTokens int64
	OldCachedAmount    float64
	NewCachedAmount    float64
	OldReasoningAmount float64
	NewReasoningAmount float64
	OldUsedAmount      float64
	NewUsedAmount      float64
	DeltaUsedAmount    float64
	NewAmount          model.Amount
}

func main() {
	flag.Parse()

	if *startDate == "" || *endDate == "" {
		log.Fatal("--start and --end are required")
	}

	if !*dryRun && !*confirm {
		log.Fatal("Must specify --confirm to apply changes (or use --dry-run)")
	}

	err := model.InitDB()
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	defer func() {
		sqlDB, _ := model.DB.DB()
		_ = sqlDB.Close()
	}()

	fmt.Println("=== Responses API Billing Repair ===")
	fmt.Printf("Time range: %s ~ %s\n", *startDate, *endDate)
	if *dryRun {
		fmt.Println("Mode: dry-run (no changes will be applied)")
	} else {
		fmt.Println("Mode: LIVE (changes will be applied)")
	}
	fmt.Println()

	stats := &RepairStats{
		ModelStats: make(map[string]*ModelRepairStats),
	}

	// Phase 1: Scan affected logs
	fmt.Println("[1/3] Scanning affected logs...")
	affectedLogs, err := scanAffectedLogs(*startDate, *endDate)
	if err != nil {
		log.Fatalf("Failed to scan logs: %v", err)
	}

	stats.TotalLogs = len(affectedLogs)
	for _, l := range affectedLogs {
		if l.RequestDetail != nil {
			stats.HasDetail++
		} else {
			stats.MissingDetail++
		}
	}

	fmt.Printf("  Total logs: %d\n", stats.TotalLogs)
	fmt.Printf("  Has request_detail: %d (%.1f%%)\n", stats.HasDetail, float64(stats.HasDetail)/float64(stats.TotalLogs)*100)
	fmt.Printf("  Missing detail: %d (%.1f%%)\n", stats.MissingDetail, float64(stats.MissingDetail)/float64(stats.TotalLogs)*100)
	fmt.Println()

	if stats.HasDetail == 0 {
		log.Fatal("No logs with request_detail found. Cannot repair.")
	}

	// Phase 2: Recompute usage and amount
	fmt.Println("[2/3] Recomputing usage and amount...")
	records := make([]*RepairRecord, 0, stats.HasDetail)

	for i, l := range affectedLogs {
		if l.RequestDetail == nil {
			stats.Skipped++
			continue
		}

		record, err := recomputeLog(l)
		if err != nil {
			log.Printf("Failed to recompute log %d: %v", l.ID, err)
			stats.Skipped++
			continue
		}

		if record == nil {
			// No change needed
			stats.Skipped++
			continue
		}

		records = append(records, record)
		stats.Processed++

		// Update stats
		stats.DeltaCachedAmount += record.NewCachedAmount - record.OldCachedAmount
		stats.DeltaReasoningAmount += record.NewReasoningAmount - record.OldReasoningAmount
		stats.DeltaUsedAmount += record.DeltaUsedAmount

		modelKey := record.Model
		if stats.ModelStats[modelKey] == nil {
			stats.ModelStats[modelKey] = &ModelRepairStats{Model: modelKey}
		}
		ms := stats.ModelStats[modelKey]
		ms.LogCount++
		ms.DeltaCachedAmount += record.NewCachedAmount - record.OldCachedAmount
		ms.DeltaReasoningAmount += record.NewReasoningAmount - record.OldReasoningAmount
		ms.DeltaUsedAmount += record.DeltaUsedAmount

		if (i+1)%100 == 0 {
			fmt.Printf("  Processed: %d / %d (%.1f%%)\r", i+1, stats.TotalLogs, float64(i+1)/float64(stats.TotalLogs)*100)
		}
	}
	fmt.Printf("  Processed: %d / %d (100%%)\n", stats.TotalLogs, stats.TotalLogs)
	fmt.Println()

	// Print summary
	fmt.Println("  Summary:")
	fmt.Printf("    Total delta_cached_amount: $%.2f\n", stats.DeltaCachedAmount)
	fmt.Printf("    Total delta_reasoning_amount: $%.2f\n", stats.DeltaReasoningAmount)
	fmt.Printf("    Total delta_used_amount: $%.2f (refund)\n", stats.DeltaUsedAmount)
	fmt.Println()

	// Top affected models
	fmt.Println("  Top affected models:")
	topModels := getTopModels(stats.ModelStats, 5)
	for _, ms := range topModels {
		fmt.Printf("    %s: $%.2f (%d logs)\n", ms.Model, ms.DeltaUsedAmount, ms.LogCount)
	}
	fmt.Println()

	// Phase 3: Apply changes or generate report
	fmt.Println("[3/3] Generating adjustment report...")
	err = writeReport(records, *outputReport)
	if err != nil {
		log.Fatalf("Failed to write report: %v", err)
	}
	fmt.Printf("  Report saved to: %s\n", *outputReport)
	fmt.Println()

	if !*dryRun {
		fmt.Println("Applying changes to database...")
		err = applyChanges(records, *batchSize)
		if err != nil {
			log.Fatalf("Failed to apply changes: %v", err)
		}
		fmt.Println("Changes applied successfully.")
	} else {
		fmt.Println("=== Dry-run completed. No data was modified. ===")
		fmt.Println("Run with --confirm to apply changes.")
	}
}

func scanAffectedLogs(start, end string) ([]*model.Log, error) {
	startTime, err := time.Parse("2006-01-02", start)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	endTime, err := time.Parse("2006-01-02", end)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}
	endTime = endTime.Add(24 * time.Hour) // Include end date

	var logs []*model.Log
	err = model.DB.
		Preload("RequestDetail").
		Where("endpoint = ?", "POST /v1/responses").
		Where("cached_tokens = 0"). // Bug symptom
		Where("created_at >= ? AND created_at < ?", startTime, endTime).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, err
}

func recomputeLog(log *model.Log) (*RepairRecord, error) {
	if log.RequestDetail == nil {
		return nil, fmt.Errorf("no request_detail")
	}

	// Extract usage using the fixed passthrough logic
	responseBytes := []byte(log.RequestDetail.ResponseBody)
	newUsage := passthrough.ExtractUsageFromBytes(responseBytes, false)

	// Check if there's actually a difference
	if newUsage.CachedTokens == log.Usage.CachedTokens &&
		newUsage.ReasoningTokens == log.Usage.ReasoningTokens {
		// No change needed
		return nil, nil
	}

	// Recompute amount using the log's original price
	newAmount := consume.CalculateAmountDetail(
		log.Code,
		newUsage,
		log.Price,
		log.ServiceTier,
	)

	record := &RepairRecord{
		LogID:              log.ID,
		RequestID:          string(log.RequestID),
		Model:              log.Model,
		CreatedAt:          log.CreatedAt,
		OldCachedTokens:    int64(log.Usage.CachedTokens),
		NewCachedTokens:    int64(newUsage.CachedTokens),
		OldReasoningTokens: int64(log.Usage.ReasoningTokens),
		NewReasoningTokens: int64(newUsage.ReasoningTokens),
		OldCachedAmount:    log.Amount.CachedAmount,
		NewCachedAmount:    newAmount.CachedAmount,
		OldReasoningAmount: log.Amount.ReasoningAmount,
		NewReasoningAmount: newAmount.ReasoningAmount,
		OldUsedAmount:      log.Amount.UsedAmount,
		NewUsedAmount:      newAmount.UsedAmount,
		DeltaUsedAmount:    newAmount.UsedAmount - log.Amount.UsedAmount,
		NewAmount:          newAmount,
	}

	return record, nil
}

func applyChanges(records []*RepairRecord, batchSize int) error {
	total := len(records)
	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}
		batch := records[i:end]

		err := model.DB.Transaction(func(tx *gorm.DB) error {
			for _, r := range batch {
				err := tx.Model(&model.Log{}).
					Where("id = ?", r.LogID).
					Updates(map[string]interface{}{
						"cached_tokens":         r.NewCachedTokens,
						"reasoning_tokens":      r.NewReasoningTokens,
						"input_amount":          r.NewAmount.InputAmount,
						"image_input_amount":    r.NewAmount.ImageInputAmount,
						"audio_input_amount":    r.NewAmount.AudioInputAmount,
						"output_amount":         r.NewAmount.OutputAmount,
						"image_output_amount":   r.NewAmount.ImageOutputAmount,
						"cached_amount":         r.NewCachedAmount,
						"cache_creation_amount": r.NewAmount.CacheCreationAmount,
						"reasoning_amount":      r.NewReasoningAmount,
						"web_search_amount":     r.NewAmount.WebSearchAmount,
						"used_amount":           r.NewUsedAmount,
					}).Error
				if err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to update batch %d-%d: %w", i, end, err)
		}

		fmt.Printf("  Updated: %d / %d (%.1f%%)\r", end, total, float64(end)/float64(total)*100)
		time.Sleep(100 * time.Millisecond) // Rate limiting
	}
	fmt.Printf("  Updated: %d / %d (100%%)\n", total, total)
	return nil
}

func writeReport(records []*RepairRecord, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// CSV header
	fmt.Fprintln(f, "log_id,request_id,model,created_at,old_cached_tokens,new_cached_tokens,old_reasoning_tokens,new_reasoning_tokens,old_cached_amount,new_cached_amount,old_reasoning_amount,new_reasoning_amount,old_used_amount,new_used_amount,delta_used_amount")

	for _, r := range records {
		fmt.Fprintf(f, "%d,%s,%s,%s,%d,%d,%d,%d,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f\n",
			r.LogID,
			r.RequestID,
			r.Model,
			r.CreatedAt.Format("2006-01-02 15:04:05"),
			r.OldCachedTokens,
			r.NewCachedTokens,
			r.OldReasoningTokens,
			r.NewReasoningTokens,
			r.OldCachedAmount,
			r.NewCachedAmount,
			r.OldReasoningAmount,
			r.NewReasoningAmount,
			r.OldUsedAmount,
			r.NewUsedAmount,
			r.DeltaUsedAmount,
		)
	}

	return nil
}

func getTopModels(modelStats map[string]*ModelRepairStats, limit int) []*ModelRepairStats {
	models := make([]*ModelRepairStats, 0, len(modelStats))
	for _, ms := range modelStats {
		models = append(models, ms)
	}

	// Simple bubble sort for top N
	for i := 0; i < len(models) && i < limit; i++ {
		for j := i + 1; j < len(models); j++ {
			if models[j].DeltaUsedAmount < models[i].DeltaUsedAmount {
				models[i], models[j] = models[j], models[i]
			}
		}
	}

	if len(models) > limit {
		models = models[:limit]
	}
	return models
}
