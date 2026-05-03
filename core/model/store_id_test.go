package model_test

import (
	"strings"
	"testing"

	"github.com/labring/aiproxy/core/model"
)

func TestHashedStoreIDDoesNotCollideOnPartSeparators(t *testing.T) {
	left := model.HashedStoreID("test", "stable", "a:b", "c")
	right := model.HashedStoreID("test", "stable", "a", "b:c")

	if left == right {
		t.Fatalf("expected distinct store ids for distinct parts, got %q", left)
	}
}

func TestHashedStoreIDKeepsPrefix(t *testing.T) {
	got := model.HashedStoreID("test", "stable", "model", "cache-key")

	if !strings.HasPrefix(got, "test:") {
		t.Fatalf("expected prefixed store id, got %q", got)
	}
}
