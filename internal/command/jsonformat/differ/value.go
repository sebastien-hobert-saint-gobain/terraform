package differ

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

// Value contains the unmarshalled generic interface{} types that are output by
// the JSON functions in the various json packages (such as jsonplan and
// jsonprovider).
//
// A Value can be converted into a change.Change, ready for rendering, with the
// ComputeChangeForAttribute, ComputeChangeForOutput, and ComputeChangeForBlock
// functions.
//
// The Before and After fields are actually go-cty values, but we cannot convert
// them directly because of the Terraform Cloud redacted endpoint. The redacted
// endpoint turns sensitive values into strings regardless of their types.
// Because of this, we cannot just do a direct conversion using the ctyjson
// package. We would have to iterate through the schema first, find the
// sensitive values and their mapped types, update the types inside the schema
// to strings, and then go back and do the overall conversion. This isn't
// including any of the more complicated parts around what happens if something
// was sensitive before and isn't sensitive after or vice versa. This would mean
// the type would need to change between the before and after value. It is in
// fact just easier to iterate through the values as generic JSON interfaces.
type Value struct {

	// BeforeExplicit matches AfterExplicit except references the Before value.
	BeforeExplicit bool

	// AfterExplicit refers to whether the After value is explicit or
	// implicit. It is explicit if it has been specified by the user, and
	// implicit if it has been set as a consequence of other changes.
	//
	// For example, explicitly setting a value to null in a list should result
	// in After being null and AfterExplicit being true. In comparison,
	// removing an element from a list should also result in After being null
	// and AfterExplicit being false. Without the explicit information our
	// functions would not be able to tell the difference between these two
	// cases.
	AfterExplicit bool

	// Before contains the value before the proposed change.
	//
	// The type of the value should be informed by the schema and cast
	// appropriately when needed.
	Before interface{}

	// After contains the value after the proposed change.
	//
	// The type of the value should be informed by the schema and cast
	// appropriately when needed.
	After interface{}

	// Unknown describes whether the After value is known or unknown at the time
	// of the plan. In practice, this means the after value should be rendered
	// simply as `(known after apply)`.
	//
	// The concrete value could be a boolean describing whether the entirety of
	// the After value is unknown, or it could be a list or a map depending on
	// the schema describing whether specific elements or attributes within the
	// value are unknown.
	Unknown interface{}

	// BeforeSensitive matches Unknown, but references whether the Before value
	// is sensitive.
	BeforeSensitive interface{}

	// AfterSensitive matches Unknown, but references whether the After value is
	// sensitive.
	AfterSensitive interface{}

	// ReplacePaths generally contains nested slices that describe paths to
	// elements or attributes that are causing the overall resource to be
	// replaced.
	ReplacePaths interface{}
}

// ValueFromJsonChange unmarshals the raw []byte values in the jsonplan.Change
// structs into generic interface{} types that can be reasoned about.
func ValueFromJsonChange(change jsonplan.Change) Value {
	return Value{
		Before:          unmarshalGeneric(change.Before),
		After:           unmarshalGeneric(change.After),
		Unknown:         unmarshalGeneric(change.AfterUnknown),
		BeforeSensitive: unmarshalGeneric(change.BeforeSensitive),
		AfterSensitive:  unmarshalGeneric(change.AfterSensitive),
		ReplacePaths:    unmarshalGeneric(change.ReplacePaths),
	}
}

// ComputeChange is a generic function that lets callers no worry about what
// type of change they are processing.
//
// It can accept blocks, attributes, go-cty types, and outputs, and will route
// the request to the appropriate function.
func (v Value) ComputeChange(changeType interface{}) change.Change {
	switch concrete := changeType.(type) {
	case *jsonprovider.Attribute:
		return v.ComputeChangeForAttribute(concrete)
	case cty.Type:
		return v.ComputeChangeForType(concrete)
	default:
		panic(fmt.Sprintf("unrecognized change type: %T", changeType))
	}
}

func (v Value) AsChange(renderer change.Renderer) change.Change {
	return change.New(renderer, v.calculateChange(), v.replacePath())
}

func (v Value) replacePath() bool {
	if replace, ok := v.ReplacePaths.(bool); ok {
		return replace
	}
	return false
}

func (v Value) calculateChange() plans.Action {
	if (v.Before == nil && !v.BeforeExplicit) && (v.After != nil || v.AfterExplicit) {
		return plans.Create
	}
	if (v.After == nil && !v.AfterExplicit) && (v.Before != nil || v.BeforeExplicit) {
		return plans.Delete
	}

	if reflect.DeepEqual(v.Before, v.After) && v.AfterExplicit == v.BeforeExplicit && v.isAfterSensitive() == v.isBeforeSensitive() {
		return plans.NoOp
	}

	return plans.Update
}

func unmarshalGeneric(raw json.RawMessage) interface{} {
	if raw == nil {
		return nil
	}

	var out interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		panic("unrecognized json type: " + err.Error())
	}
	return out
}
