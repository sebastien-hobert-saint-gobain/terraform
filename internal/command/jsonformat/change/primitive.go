package change

import (
	"fmt"
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

func (renderer primitiveRenderer) Render(result Change, indent int, opts RenderOpts) string {
	beforeValue := renderPrimitiveValue(renderer.before, renderer.t, result.indent(indent+1))
	afterValue := renderPrimitiveValue(renderer.after, renderer.t, result.indent(indent+1))

	switch result.action {
	case plans.Create:
		return fmt.Sprintf("%s%s", afterValue, result.forcesReplacement())
	case plans.Delete:
		return fmt.Sprintf("%s%s%s", beforeValue, result.nullSuffix(opts.overrideNullSuffix), result.forcesReplacement())
	case plans.NoOp:
		return fmt.Sprintf("%s%s", beforeValue, result.forcesReplacement())
	default:
		return fmt.Sprintf("%s [yellow]->[reset] %s%s", beforeValue, afterValue, result.forcesReplacement())
	}
}

func renderPrimitiveValue(value interface{}, t cty.Type, indent string) string {
	switch value.(type) {
	case nil:
		return "[dark_gray]null[reset]"
	}

	switch {
	case t == cty.String:
		str := value.(string)
		if strings.Contains(str, "\n") {
			return fmt.Sprintf("<<-\n%s\nEOT", strings.ReplaceAll(str, "\n", fmt.Sprintf("\n%s", indent)))
		} else {
			return fmt.Sprintf("\"%s\"", str)
		}
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
