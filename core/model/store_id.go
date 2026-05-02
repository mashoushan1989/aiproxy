package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	StorePrefixPromptCacheKey  = "prompt_cache_key"
	StorePrefixCacheFollow     = "cachefollow"
	StorePrefixCacheFollowUser = "cachefollow_user"
)

type CacheKeyType string

const (
	CacheKeyTypeStable CacheKeyType = "stable"
	CacheKeyTypeRecent CacheKeyType = "recent"
)

func StoreID(prefix, id string) string {
	if id == "" {
		return ""
	}

	nsPrefix := prefix + ":"
	if strings.HasPrefix(id, nsPrefix) {
		return id
	}

	return nsPrefix + id
}

func HashedStoreID(prefix string, parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	sum := sha256.Sum256(fmt.Appendf(nil, "%s", strings.Join(parts, ":")))

	return StoreID(prefix, hex.EncodeToString(sum[:]))
}

func PromptCacheStoreID(modelName, promptCacheKey string, keyType CacheKeyType) string {
	return HashedStoreID(StorePrefixPromptCacheKey, string(keyType), modelName, promptCacheKey)
}

func CacheFollowStoreID(modelName string, keyType CacheKeyType) string {
	return HashedStoreID(StorePrefixCacheFollow, string(keyType), modelName)
}

func CacheFollowUserStoreID(modelName, user string, keyType CacheKeyType) string {
	return HashedStoreID(StorePrefixCacheFollowUser, string(keyType), modelName, user)
}
