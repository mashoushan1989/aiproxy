//nolint:testpackage
package aws

import (
	"testing"

	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/stretchr/testify/assert"
)

func TestAWSModelIDMapsKnownClaudeOpus47(t *testing.T) {
	assert.Equal(
		t,
		"us.anthropic.claude-opus-4-7",
		awsModelID("claude-opus-4-7", "us-east-1"),
	)
}

func TestAWSModelIDPrefixesUnknownClaudeModel(t *testing.T) {
	assert.Equal(
		t,
		"us.anthropic.claude-custom-20260401",
		awsModelID("claude-custom-20260401", "us-east-1"),
	)
}

func TestAWSModelIDPreservesARN(t *testing.T) {
	const arn = "arn:aws:bedrock:us-east-1:123456789012:provisioned-model/test"

	assert.Equal(t, arn, awsModelID(arn, "us-east-1"))
}

func TestAWSModelIDFromMetaUsesActualModel(t *testing.T) {
	m := &meta.Meta{
		OriginModel: "claude-opus-4-7",
		ActualModel: "claude-3-haiku-20240307",
	}

	assert.Equal(
		t,
		"us.anthropic.claude-3-haiku-20240307-v1:0",
		awsModelIDFromMeta(m, "us-east-1"),
	)
}

func TestAWSModelIDFromMetaPreservesActualARN(t *testing.T) {
	const arn = "arn:aws:bedrock:us-east-1:123456789012:provisioned-model/test"

	m := &meta.Meta{
		OriginModel: "claude-opus-4-7",
		ActualModel: arn,
	}

	assert.Equal(t, arn, awsModelIDFromMeta(m, "us-east-1"))
}
