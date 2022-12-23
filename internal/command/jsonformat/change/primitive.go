package change

import (
	"fmt"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/jsonformat/list"
	"github.com/zclconf/go-cty/cty"
	"strings"

	"github.com/hashicorp/terraform/internal/plans"
)

func Primitive(before, after interface{}, t cty.Type) Renderer {
	return &primitiveRenderer{
		before: before,
		after:  after,
		t:      t,
	}
}

type primitiveRenderer struct {
	NoWarningsRenderer

	before interface{}
	after  interface{}
	t      cty.Type
}

func (renderer primitiveRenderer) Render(change Change, indent int, opts RenderOpts) string {
	if renderer.t == cty.String {
		return renderer.renderStringValue(change, indent+1, opts)
	}

	beforeValue := renderPrimitiveValue(renderer.before, renderer.t)
	afterValue := renderPrimitiveValue(renderer.after, renderer.t)

	switch change.action {
	case plans.Create:
		return fmt.Sprintf("%s%s", afterValue, change.forcesReplacement())
	case plans.Delete:
		return fmt.Sprintf("%s%s%s", beforeValue, change.nullSuffix(opts.overrideNullSuffix), change.forcesReplacement())
	case plans.NoOp:
		return fmt.Sprintf("%s%s", beforeValue, change.forcesReplacement())
	default:
		return fmt.Sprintf("%s [yellow]->[reset] %s%s", beforeValue, afterValue, change.forcesReplacement())
	}
}

func renderPrimitiveValue(value interface{}, t cty.Type) string {
	switch value.(type) {
	case nil:
		return "[dark_gray]null[reset]"
	}

	switch {
	case t == cty.Bool:
		if value.(bool) {
			return "true"
		}
		return "false"
	case t == cty.Number:
		return fmt.Sprintf("%g", value)
	default:
		panic("unrecognized primitive type: " + t.FriendlyName())
	}
}

func (renderer primitiveRenderer) renderStringValue(change Change, indent int, opts RenderOpts) string {

	var lines []string

	switch change.action {
	case plans.Create, plans.NoOp:
		str, multiline := concretizeStringValue(renderer.after)
		if !multiline {
			return fmt.Sprintf("%s%s", str, change.forcesReplacement())
		}
		lines = strings.Split(strings.ReplaceAll(str, "\n", fmt.Sprintf("\n%s%s ", change.indent(indent), change.emptySymbol())), "\n")
		lines[0] = fmt.Sprintf("%s%s %s", change.indent(indent), change.emptySymbol(), lines[0]) // We have to manually add the indent for the first line.
	case plans.Delete:
		str, multiline := concretizeStringValue(renderer.before)
		if !multiline {
			return fmt.Sprintf("%s%s%s", str, change.nullSuffix(opts.overrideNullSuffix), change.forcesReplacement())
		}
		lines = strings.Split(strings.ReplaceAll(str, "\n", fmt.Sprintf("\n%s%s ", change.indent(indent), change.emptySymbol())), "\n")
		lines[0] = fmt.Sprintf("%s%s %s", change.indent(indent), change.emptySymbol(), lines[0]) // We have to manually add the indent for the first line.
	default:
		beforeStr, beforeMulti := concretizeStringValue(renderer.before)
		afterStr, afterMulti := concretizeStringValue(renderer.after)
		if !beforeMulti && !afterMulti {
			return fmt.Sprintf("%s [yellow]->[reset] %s%s", beforeStr, afterStr, change.forcesReplacement())
		}

		beforeLines := strings.Split(beforeStr, "\n")
		afterLines := strings.Split(afterStr, "\n")

		list.Process(beforeLines, afterLines, false, func(beforeIx, afterIx int) {
			if beforeIx < 0 || beforeIx >= len(beforeLines) {
				lines = append(lines, fmt.Sprintf("%s%s %s", change.indent(indent), format.DiffActionSymbol(plans.Create), afterLines[afterIx]))
				return
			}

			if afterIx < 0 || afterIx >= len(afterLines) {
				lines = append(lines, fmt.Sprintf("%s%s %s", change.indent(indent), format.DiffActionSymbol(plans.Delete), beforeLines[beforeIx]))
				return
			}

			lines = append(lines, fmt.Sprintf("%s%s %s", change.indent(indent), change.emptySymbol(), beforeLines[beforeIx]))
		})
	}

	return fmt.Sprintf("<<-EOT%s\n%s\n%sEOT%s",
		change.forcesReplacement(),
		strings.Join(lines, "\n"),
		change.indent(indent),
		change.nullSuffix(opts.overrideNullSuffix))
}

func concretizeStringValue(value interface{}) (string, bool) {
	switch value.(type) {
	case nil:
		return "null", false
	default:
		str := value.(string)
		if strings.Contains(str, "\n") {
			return strings.TrimSpace(str), true
		}
		return fmt.Sprintf("\"%s\"", str), false
	}
}
