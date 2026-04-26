//go:build enterprise

package synccommon

import (
	"fmt"
	"slices"

	"github.com/labring/aiproxy/core/model"
	"gorm.io/gorm"
)

// AddModelToPeerChannels appends modelName to all status=1 channels of
// channelType that share at least one Set with originChannelID's channel.
//
// Used by autodiscover hooks to propagate a newly-discovered model to peer
// channels in the same routing scope (e.g. domestic→domestic, overseas→
// overseas) without leaking across sets. Empty Sets is treated as the default
// set per Channel.GetSets().
//
// Skips channels that already contain the model. Best-effort: per-channel
// errors are returned aggregated; partial success is the common case when one
// channel save races with another writer.
func AddModelToPeerChannels(
	db *gorm.DB,
	originChannelID int,
	channelType model.ChannelType,
	modelName string,
) error {
	var origin model.Channel
	if err := db.Select("id, sets").First(&origin, originChannelID).Error; err != nil {
		return fmt.Errorf("read origin channel %d: %w", originChannelID, err)
	}

	originSets := origin.GetSets()

	// We only need id, sets and models from peers. Selecting these explicitly
	// keeps the in-memory copy small AND — combined with the column-scoped
	// Updates() below — avoids the read-modify-write race where a concurrent
	// writer's edits to other columns (configs, status, key, etc.) would be
	// clobbered by db.Save() blindly writing back the whole row.
	var peers []model.Channel
	if err := db.Select("id, sets, models").
		Where("type = ? AND status = ?", channelType, 1).
		Find(&peers).Error; err != nil {
		return fmt.Errorf("find peer channels of type %d: %w", channelType, err)
	}

	var firstErr error

	for i := range peers {
		if !SetsIntersect(originSets, peers[i].GetSets()) {
			continue
		}

		if slices.Contains(peers[i].Models, modelName) {
			continue
		}

		newModels := append(slices.Clone(peers[i].Models), modelName)

		if err := db.Model(&model.Channel{}).
			Where("id = ?", peers[i].ID).
			Update("models", newModels).Error; err != nil && firstErr == nil {
			firstErr = fmt.Errorf("save peer %d: %w", peers[i].ID, err)
		}
	}

	return firstErr
}

// SetsIntersect reports whether two channel-set slices share any element.
// Used by autodiscover peer-fanout to scope propagation to channels in the
// same routing set as the origin.
func SetsIntersect(a, b []string) bool {
	for _, x := range a {
		if slices.Contains(b, x) {
			return true
		}
	}

	return false
}
