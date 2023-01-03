package change

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/plans"
)

func Computed(before Change) Renderer {
	return &computedRenderer{
		before: before,
	}
}

type computedRenderer struct {
	NoWarningsRenderer

	before Change
}

func (renderer computedRenderer) Render(change Change, indent int, opts RenderOpts) string {
	if change.action == plans.Create {
		return "(known after apply)"
	}

	// Never render null suffix for children of computed changes.
	opts.overrideNullSuffix = true
	return fmt.Sprintf("%s -> (known after apply)", renderer.before.Render(indent, opts))
}

func (renderer computedRenderer) ContainsSensitive() bool {
	if renderer.before.action == plans.NoOp {
		return false
	}
	return renderer.before.ContainsSensitive()
}
