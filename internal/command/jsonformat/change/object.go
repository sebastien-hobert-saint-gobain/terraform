package change

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/terraform/internal/command/format"
	"sort"

	"github.com/hashicorp/terraform/internal/plans"
)

func Object(attributes map[string]Change) Renderer {
	return &objectRenderer{
		attributes:         attributes,
		overrideNullSuffix: true,
	}
}

func NestedObject(attributes map[string]Change) Renderer {
	return &objectRenderer{
		attributes:         attributes,
		overrideNullSuffix: false,
	}
}

type objectRenderer struct {
	NoWarningsRenderer

	attributes         map[string]Change
	overrideNullSuffix bool
}

func (renderer objectRenderer) Render(change Change, indent int, opts RenderOpts) string {
	if len(renderer.attributes) == 0 {
		return fmt.Sprintf("{}%s%s", change.nullSuffix(opts.overrideNullSuffix), change.forcesReplacement())
	}

	attributeOpts := opts.Clone()
	attributeOpts.overrideNullSuffix = renderer.overrideNullSuffix

	maximumKeyLen := 0
	escapedKeys := make(map[string]string)
	var keys []string
	for key := range renderer.attributes {
		keys = append(keys, key)

		esc := change.escapeAttributeName(key)
		if len(esc) > maximumKeyLen {
			maximumKeyLen = len(esc)
		}
		escapedKeys[key] = esc
	}
	sort.Strings(keys)

	unchangedAttributes := 0
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("{%s\n", change.forcesReplacement()))
	for _, key := range keys {
		attribute := renderer.attributes[key]

		if !importantAttribute(key) && attribute.action == plans.NoOp && !opts.showUnchangedChildren {
			// Don't render NoOp operations when we are compact display.
			unchangedAttributes++
			continue
		}

		for _, warning := range attribute.Warnings(indent + 1) {
			buf.WriteString(fmt.Sprintf("%s%s\n", change.indent(indent+1), warning))
		}

		if attribute.action == plans.NoOp {
			buf.WriteString(fmt.Sprintf("%s%s %-*s = %s\n", change.indent(indent+1), change.emptySymbol(), maximumKeyLen, escapedKeys[key], attribute.Render(indent+1, attributeOpts)))
		} else {
			buf.WriteString(fmt.Sprintf("%s%s %-*s = %s\n", change.indent(indent+1), format.DiffActionSymbol(attribute.action), maximumKeyLen, escapedKeys[key], attribute.Render(indent+1, attributeOpts)))
		}
	}

	if unchangedAttributes > 0 {
		buf.WriteString(fmt.Sprintf("%s%s %s\n", change.indent(indent+1), change.emptySymbol(), change.unchanged("attribute", unchangedAttributes)))
	}

	buf.WriteString(fmt.Sprintf("%s%s }%s", change.indent(indent), change.emptySymbol(), change.nullSuffix(opts.overrideNullSuffix)))
	return buf.String()
}

func (renderer objectRenderer) ContainsSensitive() bool {
	for _, attribute := range renderer.attributes {
		if attribute.ContainsSensitive() {
			return true
		}
	}
	return false
}
