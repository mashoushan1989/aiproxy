package task

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/balance"
	"github.com/labring/aiproxy/core/common/consume"
	"github.com/labring/aiproxy/core/common/notify"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptors"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	asyncUsagePollInterval    = time.Second * 3
	asyncUsageProcessingLease = time.Minute * 3
	asyncUsageBatchSize       = 50
	asyncUsageConcurrency     = 10
	asyncUsageMaxRetry        = 10
)

func AsyncUsagePollTask(ctx context.Context) {
	ticker := time.NewTicker(asyncUsagePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for {
				fullBatch := processAsyncUsages(ctx)
				if !fullBatch {
					break
				}

				select {
				case <-ctx.Done():
					return
				default:
				}
			}
		}
	}
}

func processAsyncUsages(ctx context.Context) bool {
	infos, err := model.GetPendingAsyncUsages(asyncUsageBatchSize)
	if err != nil {
		notify.ErrorThrottle(
			"asyncUsagePoll",
			time.Minute*5,
			"get pending async usages failed",
			err.Error(),
		)

		return false
	}

	if len(infos) == 0 {
		return false
	}

	claimedInfos := make([]*model.AsyncUsageInfo, 0, len(infos))
	for _, info := range infos {
		claimed, err := claimAsyncUsage(info)
		if err != nil {
			notify.ErrorThrottle(
				"asyncUsageClaim",
				time.Minute*5,
				"claim async usage failed",
				err.Error(),
			)

			continue
		}

		if claimed {
			claimedInfos = append(claimedInfos, info)
		}
	}

	if len(claimedInfos) == 0 {
		return len(infos) == asyncUsageBatchSize
	}

	sem := make(chan struct{}, asyncUsageConcurrency)

	var wg sync.WaitGroup
	for _, info := range claimedInfos {
		select {
		case <-ctx.Done():
			wg.Wait()
			return false
		case sem <- struct{}{}:
		}

		wg.Add(1)

		go func(info *model.AsyncUsageInfo) {
			defer wg.Done()
			defer func() {
				<-sem
			}()

			processOneAsyncUsage(ctx, info)
		}(info)
	}

	wg.Wait()

	return len(infos) == asyncUsageBatchSize
}

func claimAsyncUsage(info *model.AsyncUsageInfo) (bool, error) {
	now := time.Now()
	token := common.ShortUUID()
	leaseUntil := now.Add(asyncUsageProcessingLease)

	claimed, err := model.TryClaimAsyncUsageInfo(info, token, leaseUntil, now)
	if err != nil || !claimed {
		return claimed, err
	}

	log.Debugf(
		"async usage poll: claimed id=%d request_id=%s upstream_id=%s lease_until=%s",
		info.ID,
		info.RequestID,
		info.UpstreamID,
		leaseUntil.Format(time.RFC3339),
	)

	return true, nil
}

func processOneAsyncUsage(ctx context.Context, info *model.AsyncUsageInfo) {
	stopRenew := startAsyncUsageClaimRenewal(ctx, info)
	defer stopRenew()

	channel, err := model.GetChannelByID(info.ChannelID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			scheduleAsyncUsageRetry(info, fmt.Errorf("get channel: %w", err))
			return
		}

		markAsyncUsageFailed(info, "channel not found: "+err.Error())
		return
	}

	a, ok := adaptors.GetAdaptor(channel.Type)
	if !ok {
		markAsyncUsageFailed(info, fmt.Sprintf("adaptor not found for channel type %d", channel.Type))
		return
	}

	fetcher, ok := a.(adaptor.AsyncUsageFetcher)
	if !ok {
		markAsyncUsageFailed(info, "adaptor does not support async usage fetching")
		return
	}

	usage, completed, err := fetcher.FetchAsyncUsage(ctx, channel, info)
	if err != nil {
		if completed {
			markAsyncUsageFailed(info, err.Error())
			return
		}

		scheduleAsyncUsageRetry(info, err)
		return
	}

	if !completed {
		touchAsyncUsagePollCursor(info)
		return
	}

	if err := completeAsyncUsage(ctx, info, usage); err != nil {
		scheduleAsyncUsageRetry(info, fmt.Errorf("complete failed: %w", err))
	}
}

func startAsyncUsageClaimRenewal(ctx context.Context, info *model.AsyncUsageInfo) func() {
	done := make(chan struct{})
	interval := asyncUsageProcessingLease / 3

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case <-ticker.C:
				leaseUntil := time.Now().Add(asyncUsageProcessingLease)

				renewed, err := model.RenewAsyncUsageClaim(
					info.ID,
					info.ProcessingToken,
					leaseUntil,
				)
				if err != nil {
					notify.ErrorThrottle(
						"asyncUsageRenewClaim",
						time.Minute*5,
						"renew async usage claim failed",
						err.Error(),
					)

					continue
				}

				if !renewed {
					log.Debugf(
						"async usage poll: claim lost id=%d request_id=%s upstream_id=%s",
						info.ID,
						info.RequestID,
						info.UpstreamID,
					)

					return
				}

				info.NextPollAt = leaseUntil
			}
		}
	}()

	return func() {
		close(done)
	}
}

func completeAsyncUsage(ctx context.Context, info *model.AsyncUsageInfo, usage model.Usage) error {
	price := info.Price

	amount := consume.CalculateAmountDetail(
		http.StatusOK,
		usage,
		price,
		info.ServiceTier,
	)

	if amount.UsedAmount > 0 && !info.BalanceConsumed {
		if err := consumeAsyncUsageGroupBalance(ctx, info, amount.UsedAmount); err != nil {
			recordAsyncUsageConsumeError(info, amount.UsedAmount, err)
			return fmt.Errorf("consume async usage balance: %w", err)
		}

		info.BalanceConsumed = true
		if err := model.MarkAsyncUsageBalanceConsumed(info); err != nil {
			notify.ErrorThrottle(
				"asyncUsageMarkBalanceConsumed",
				time.Minute*5,
				"mark async usage balance consumed failed",
				err.Error(),
			)
		}
	}

	summaryUsage := usage
	summaryAmount := amount
	if deltaUsage, deltaAmount, err := model.UpdateLogUsageByRequestIDDelta(
		info.RequestID,
		usage,
		amount,
	); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("update async usage log: %w", err)
		}
	} else {
		summaryUsage = deltaUsage
		summaryAmount = deltaAmount
	}

	if !asyncUsageCompletionDeltaIsZero(summaryUsage, summaryAmount) {
		model.BatchUpdateSummaryOnlyUsage(
			time.Now(),
			info.RequestAt,
			info.GroupID,
			info.ChannelID,
			info.Model,
			info.TokenID,
			info.TokenName,
			summaryUsage,
			summaryAmount,
			info.ServiceTier,
			model.IsClaudeLongContextSummary(info.Model, summaryUsage),
		)
	}

	info.Status = model.AsyncUsageStatusCompleted
	info.Usage = usage
	info.Amount = amount
	info.Error = ""

	completed, err := model.CompleteClaimedAsyncUsageInfo(info, usage, amount)
	if err != nil {
		return fmt.Errorf("update async usage info: %w", err)
	}

	if !completed {
		return errors.New("async usage claim lost")
	}

	return nil
}

func asyncUsageCompletionDeltaIsZero(usage model.Usage, amount model.Amount) bool {
	return usage == (model.Usage{}) && amount == (model.Amount{})
}

func scheduleAsyncUsageRetry(info *model.AsyncUsageInfo, err error) {
	info.RetryCount++
	info.Error = err.Error()
	info.NextPollAt = time.Now().Add(model.AsyncUsageBackoffDelay(info.RetryCount))

	if info.RetryCount >= asyncUsageMaxRetry {
		markAsyncUsageFailed(info, "max retry exceeded: "+err.Error())
		return
	}

	if updateErr := model.RetryClaimedAsyncUsageInfo(info); updateErr != nil {
		notify.ErrorThrottle(
			"asyncUsageUpdateRetry",
			time.Minute*5,
			"update async usage retry failed",
			updateErr.Error(),
		)
	}
}

func touchAsyncUsagePollCursor(info *model.AsyncUsageInfo) {
	info.Error = ""
	info.NextPollAt = time.Now().Add(model.AsyncUsageDefaultPollDelay)

	if err := model.TouchClaimedAsyncUsageInfo(info); err != nil {
		notify.ErrorThrottle(
			"asyncUsageTouchPending",
			time.Minute*5,
			"touch pending async usage failed",
			err.Error(),
		)
	}
}

func consumeAsyncUsageGroupBalance(
	ctx context.Context,
	info *model.AsyncUsageInfo,
	amount float64,
) error {
	if balance.Default == nil || info.GroupID == "" {
		return nil
	}

	group, err := model.CacheGetGroup(info.GroupID)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}

	_, consumer, err := balance.Default.GetGroupRemainBalance(ctx, *group)
	if err != nil {
		return fmt.Errorf("get group balance: %w", err)
	}

	if consumer == nil {
		return nil
	}

	if idempotentConsumer, ok := consumer.(balance.IdempotentPostGroupConsumer); ok {
		_, err = idempotentConsumer.PostGroupConsumeWithKey(
			ctx,
			info.TokenName,
			amount,
			asyncUsageBalanceIdempotencyKey(info),
		)
	} else {
		_, err = consumer.PostGroupConsume(ctx, info.TokenName, amount)
	}

	return err
}

func asyncUsageBalanceIdempotencyKey(info *model.AsyncUsageInfo) string {
	if info == nil {
		return ""
	}

	return fmt.Sprintf("async_usage:%d:%s", info.ID, info.RequestID)
}

func recordAsyncUsageConsumeError(info *model.AsyncUsageInfo, amount float64, err error) {
	if err := model.CreateConsumeError(
		info.RequestID,
		info.RequestAt,
		info.GroupID,
		info.TokenName,
		info.Model,
		err.Error(),
		amount,
		info.TokenID,
	); err != nil {
		log.Error("failed to create async usage consume error: " + err.Error())
	}
}

func markAsyncUsageFailed(info *model.AsyncUsageInfo, errMsg string) {
	info.Status = model.AsyncUsageStatusFailed
	info.Error = errMsg

	updated, err := model.FailClaimedAsyncUsageInfo(info)
	if err != nil {
		notify.ErrorThrottle(
			"asyncUsageMarkFailed",
			time.Minute*5,
			"mark async usage failed",
			err.Error(),
		)

		return
	}

	if !updated {
		return
	}

	if err := model.IgnoreNotFound(
		model.UpdateLogAsyncUsageStatusByRequestID(info.RequestID, model.AsyncUsageStatusFailed),
	); err != nil {
		notify.ErrorThrottle(
			"asyncUsageUpdateLogStatus",
			time.Minute*5,
			"update async usage log status failed",
			err.Error(),
		)
	}
}
