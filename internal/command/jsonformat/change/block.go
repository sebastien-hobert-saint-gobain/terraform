package change

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/plans"
)

var (
	importantAttributes = []string{
		"id",
		"name",
		"tags",
	}
)

func importantAttribute(attr string) bool {
	for _, attribute := range importantAttributes {
		if attribute == attr {
			return true
		}
	}
	return false
}

func Block(attributes map[string]Change, blocks map[string][]Change, mapBlocks map[string]map[string]Change) Renderer {
	return &blockRenderer{
		attributes: attributes,
		blocks:     blocks,
		mapBlocks:  mapBlocks,
	}
}

type blockRenderer struct {
	NoWarningsRenderer

	attributes map[string]Change
	blocks     map[string][]Change
	mapBlocks  map[string]map[string]Change
}

func (renderer blockRenderer) Render(change Change, indent int, opts RenderOpts) string {
	if len(renderer.attributes) == 0 && len(renderer.blocks) == 0 {
		return fmt.Sprintf("{}%s", change.forcesReplacement())
	}

	if opts.elideSensitiveBlocks && renderer.ContainsSensitive() {
		indent := fmt.Sprintf("%s%s ", change.indent(indent), change.emptySymbol())
		return fmt.Sprintf("{%s\n%s  # At least one attribute in this block is (or was) sensitive,\n%s  # so its contents will not be displayed\n%s}", change.forcesReplacement(), indent, indent, indent)
	}

	unchangedAttributes := 0
	unchangedBlocks := 0

	maximumKeyLen := 0
	escapedKeys := make(map[string]string)
	var attributeKeys []string
	for key := range renderer.attributes {
		attributeKeys = append(attributeKeys, key)

		esc := change.escapeAttributeName(key)
		if len(esc) > maximumKeyLen {
			maximumKeyLen = len(esc)
		}
		escapedKeys[key] = esc
	}
	sort.Strings(attributeKeys)

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("{%s\n", change.forcesReplacement()))
	for _, key := range attributeKeys {
		attribute := renderer.attributes[key]
		if !importantAttribute(key) && (attribute.action == plans.NoOp && !opts.showUnchangedChildren) {
			unchangedAttributes++
			continue
		}

		for _, warning := range attribute.Warnings(indent + 1) {
			buf.WriteString(fmt.Sprintf("%s%s\n", change.indent(indent+1), warning))
		}

		if attribute.action == plans.NoOp {
			opts := opts.Clone()
			opts.showUnchangedChildren = true
			buf.WriteString(fmt.Sprintf("%s%s %-*s = %s\n", change.indent(indent+1), change.emptySymbol(), maximumKeyLen, escapedKeys[key], attribute.Render(indent+1, opts)))
		} else {
			buf.WriteString(fmt.Sprintf("%s%s %-*s = %s\n", change.indent(indent+1), format.DiffActionSymbol(attribute.action), maximumKeyLen, escapedKeys[key], attribute.Render(indent+1, opts)))
		}
	}

	if unchangedAttributes > 0 {
		buf.WriteString(fmt.Sprintf("%s%s %s\n", change.indent(indent+1), change.emptySymbol(), change.unchanged("attribute", unchangedAttributes)))
	}

	blockOpts := opts.Clone()
	blockOpts.elideSensitiveBlocks = true

	var blockKeys []string
	for key := range renderer.blocks {
		blockKeys = append(blockKeys, key)
	}
	for key := range renderer.mapBlocks {
		blockKeys = append(blockKeys, key)
	}
	sort.Strings(blockKeys)

	for _, key := range blockKeys {
		if blocks, ok := renderer.blocks[key]; ok {
			foundChangedBlock := false
			for _, block := range blocks {
				if block.action == plans.NoOp && !opts.showUnchangedChildren {
					unchangedBlocks++
					continue
				}

				if !foundChangedBlock && len(renderer.attributes) > 0 {
					buf.WriteString("\n")
					foundChangedBlock = true
				}

				for _, warning := range block.Warnings(indent + 1) {
					buf.WriteString(fmt.Sprintf("%s%s\n", change.indent(indent+1), warning))
				}
				buf.WriteString(fmt.Sprintf("%s%s %s %s\n", change.indent(indent+1), format.DiffActionSymbol(block.action), change.escapeAttributeName(key), block.Render(indent+1, blockOpts)))
			}
		}

		if blocks, ok := renderer.mapBlocks[key]; ok {
			foundChangedBlock := false

			var sortedBlocks []string
			for mapKey := range blocks {
				sortedBlocks = append(sortedBlocks, mapKey)
			}
			sort.Strings(sortedBlocks)

			for _, mapKey := range sortedBlocks {
				block := blocks[mapKey]
				if block.action == plans.NoOp && !opts.showUnchangedChildren {
					unchangedBlocks++
					continue
				}

				if !foundChangedBlock && len(renderer.attributes) > 0 {
					buf.WriteString("\n")
					foundChangedBlock = true
				}

				for _, warning := range block.Warnings(indent + 1) {
					buf.WriteString(fmt.Sprintf("%s%s\n", change.indent(indent+1), warning))
				}
				buf.WriteString(fmt.Sprintf("%s%s %s \"%s\" %s\n", change.indent(indent+1), format.DiffActionSymbol(block.action), change.escapeAttributeName(key), mapKey, block.Render(indent+1, blockOpts)))
			}
		}
	}

	if unchangedBlocks > 0 {
		buf.WriteString(fmt.Sprintf("\n%s%s %s\n", change.indent(indent+1), change.emptySymbol(), change.unchanged("block", unchangedBlocks)))
	}

	buf.WriteString(fmt.Sprintf("%s%s }", change.indent(indent), change.emptySymbol()))
	return buf.String()
}

func (renderer blockRenderer) ContainsSensitive() bool {
	for _, attribute := range renderer.attributes {
		if attribute.ContainsSensitive() {
			return true
		}
	}
	return false
}
