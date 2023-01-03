package differ

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

func (v Value) computeChangeForBlock(block *jsonprovider.Block) change.Change {
	current := v.getDefaultActionForIteration()

	blockValue := v.asMap()

	attributes := make(map[string]change.Change)
	for key, attr := range block.Attributes {
		childValue := blockValue.getChild(key, false)

		// Empty strings in blocks should be considered null for legacy reasons.
		// The SDK doesn't support null strings yet, so we work around this now.
		if before, ok := childValue.Before.(string); ok && len(before) == 0 {
			childValue.Before = nil
		}
		if after, ok := childValue.After.(string); ok && len(after) == 0 {
			childValue.After = nil
		}

		// Always treat changes to blocks as implicit.
		childValue.BeforeExplicit = false
		childValue.AfterExplicit = false

		childChange := childValue.ComputeChange(attr)

		if childChange.GetAction() == plans.NoOp && childValue.Before == nil && childValue.After == nil {
			// Don't record nil values at all in blocks.
			continue
		}

		attributes[key] = childChange
		current = compareActions(current, childChange.GetAction())
	}

	blocks := make(map[string][]change.Change)
	mapBlocks := make(map[string]map[string]change.Change)
	for key, blockType := range block.BlockTypes {
		childValue := blockValue.getChild(key, false)

		var next plans.Action
		var add func()

		switch blockType.NestingMode {
		case "set":
			var childChanges []change.Change
			childChanges, next = childValue.computeBlockChangesAsSet(blockType.Block)
			add = func() {
				blocks[key] = childChanges
			}
		case "list":
			var childChanges []change.Change
			childChanges, next = childValue.computeBlockChangesAsList(blockType.Block)
			add = func() {
				blocks[key] = childChanges
			}
		case "map":
			var childChanges map[string]change.Change
			childChanges, next = childValue.computeBlockChangesAsMap(blockType.Block)
			add = func() {
				mapBlocks[key] = childChanges
			}
		case "single", "group":
			ch := childValue.ComputeChange(blockType.Block)
			next = ch.GetAction()
			add = func() {
				blocks[key] = []change.Change{ch}
			}
		default:
			panic("unrecognized nesting mode: " + blockType.NestingMode)
		}

		if next == plans.NoOp && childValue.Before == nil && childValue.After == nil {
			// Don't record nil values at all in blocks.
			continue
		}
		add()
		current = compareActions(current, next)
	}

	return change.New(change.Block(attributes, blocks, mapBlocks), current, v.replacePath())
}
