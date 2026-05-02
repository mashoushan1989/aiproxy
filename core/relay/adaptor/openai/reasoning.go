package openai

import (
	"errors"

	"github.com/bytedance/sonic/ast"
	"github.com/labring/aiproxy/core/relay/meta"
)

// StreamReasoningToReasoningContentPreHandler rewrites
// choices.[*].delta.reasoning -> choices.[*].delta.reasoning_content.
func StreamReasoningToReasoningContentPreHandler(_ *meta.Meta, node *ast.Node) error {
	choicesNode := node.Get("choices")

	nodes, err := choicesNode.ArrayUseNode()
	if err != nil {
		return err
	}

	for index, choice := range nodes {
		deltaNode := choice.Get("delta")

		reasoningString, err := deltaNode.Get("reasoning").String()
		if err != nil {
			if errors.Is(err, ast.ErrNotExist) {
				continue
			}
			return err
		}

		if _, err = deltaNode.Set("reasoning_content", ast.NewString(reasoningString)); err != nil {
			return err
		}

		if _, err = deltaNode.Unset("reasoning"); err != nil {
			return err
		}

		if _, err = choicesNode.SetByIndex(index, choice); err != nil {
			return err
		}
	}

	return nil
}

// ReasoningToReasoningContentPreHandler rewrites
// choices.[*].message.reasoning -> choices.[*].message.reasoning_content.
func ReasoningToReasoningContentPreHandler(_ *meta.Meta, node *ast.Node) error {
	choicesNode := node.Get("choices")

	nodes, err := choicesNode.ArrayUseNode()
	if err != nil {
		return err
	}

	for index, choice := range nodes {
		messageNode := choice.Get("message")

		reasoningString, err := messageNode.Get("reasoning").String()
		if err != nil {
			if errors.Is(err, ast.ErrNotExist) {
				continue
			}
			return err
		}

		if _, err = messageNode.Set("reasoning_content", ast.NewString(reasoningString)); err != nil {
			return err
		}

		if _, err = messageNode.Unset("reasoning"); err != nil {
			return err
		}

		if _, err = choicesNode.SetByIndex(index, choice); err != nil {
			return err
		}
	}

	return nil
}
