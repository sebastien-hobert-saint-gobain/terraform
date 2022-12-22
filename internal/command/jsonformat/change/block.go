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

func Block(attributes map[string]Change, blocks map[string][]Change) Renderer {
	maximumKeyLen := 0
	for key := range attributes {
		if len(key) > maximumKeyLen {
			maximumKeyLen = len(key)
		}
	}

	return &blockRenderer{
		attributes:    attributes,
		blocks:        blocks,
		maximumKeyLen: maximumKeyLen,
	}
}

type blockRenderer struct {
	NoWarningsRenderer

	attributes    map[string]Change
	blocks        map[string][]Change
	maximumKeyLen int
}

func (renderer blockRenderer) Render(change Change, indent int, opts RenderOpts) string {
	unchangedAttributes := 0
	unchangedBlocks := 0

	var attributeKeys []string
	for key := range renderer.attributes {
		attributeKeys = append(attributeKeys, key)
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
			buf.WriteString(fmt.Sprintf("%s%s %-*s = %s\n", change.indent(indent+1), change.emptySymbol(), renderer.maximumKeyLen, key, attribute.Render(indent+1, opts)))
		} else {
			buf.WriteString(fmt.Sprintf("%s%s %-*s = %s\n", change.indent(indent+1), format.DiffActionSymbol(attribute.action), renderer.maximumKeyLen, key, attribute.Render(indent+1, opts)))
		}
	}

	if unchangedAttributes > 0 {
		buf.WriteString(fmt.Sprintf("%s%s %s\n", change.indent(indent+1), change.emptySymbol(), change.unchanged("attribute", unchangedAttributes)))
	}

	var blockKeys []string
	for key := range renderer.blocks {
		blockKeys = append(blockKeys, key)
	}
	sort.Strings(blockKeys)

	printedAnyBlocks := false
	for _, key := range blockKeys {
		blocks := renderer.blocks[key]

		foundChangedBlock := false
		for _, block := range blocks {
			if block.action == plans.NoOp && !opts.showUnchangedChildren {
				unchangedBlocks++
				continue
			}

			if !foundChangedBlock && len(renderer.attributes) > 0 {
				buf.WriteString("\n")
				foundChangedBlock = true
				printedAnyBlocks = true
			}

			for _, warning := range block.Warnings(indent + 1) {
				buf.WriteString(fmt.Sprintf("%s%s\n", change.indent(indent+1), warning))
			}
			buf.WriteString(fmt.Sprintf("%s%s %s %s\n", change.indent(indent+1), format.DiffActionSymbol(block.action), key, block.Render(indent+1, opts)))
		}
	}

	if unchangedBlocks > 0 {
		if !printedAnyBlocks {
			buf.WriteString("\n")
		}
		buf.WriteString(fmt.Sprintf("%s%s %s\n", change.indent(indent+1), change.emptySymbol(), change.unchanged("block", unchangedBlocks)))
	}

	buf.WriteString(fmt.Sprintf("%s%s }", change.indent(indent), change.emptySymbol()))
	return buf.String()
}
