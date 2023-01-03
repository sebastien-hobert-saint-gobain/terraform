package differ

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
)

func (v Value) checkForComputed(changeType interface{}) (change.Change, bool) {
	unknown := v.isUnknown()

	if !unknown {
		return change.Change{}, false
	}

	// No matter what we do here, we want to treat the after value as explicit.
	// This is because it is going to be null in the value, and we don't want
	// the functions in this package to assume this means it has been deleted.
	v.AfterExplicit = true

	if v.Before == nil {
		return v.AsChange(change.Computed(change.Change{})), true
	}

	// If we get here, then we have a before value. We're going to model a
	// delete operation and our renderer later can render the overall change
	// accurately.

	var childUnknown interface{}
	if attribute, ok := changeType.(*jsonprovider.Attribute); ok && attribute.AttributeNestedType != nil {
		// Small bit of a special case here, when processing a nested type we
		// want the attributes to show as being computed instead of deleted.

		unknown := make(map[string]interface{})
		for key := range attribute.AttributeNestedType.Attributes {
			unknown[key] = true
		}
		childUnknown = unknown
	}

	if attributes, ok := changeType.(map[string]*jsonprovider.Attribute); ok {
		// Same bit of logic here, but in case we are processing the nested
		// attributes directly.

		unknown := make(map[string]interface{})
		for key := range attributes {
			unknown[key] = true
		}
		childUnknown = unknown
	}

	beforeValue := Value{
		Before:          v.Before,
		BeforeSensitive: v.BeforeSensitive,
		Unknown:         childUnknown,
	}
	return v.AsChange(change.Computed(beforeValue.ComputeChange(changeType))), true
}

func (v Value) isUnknown() bool {
	if unknown, ok := v.Unknown.(bool); ok {
		return unknown
	}
	return false
}

func anyUnknown(value interface{}) bool {
	switch concrete := value.(type) {
	case bool:
		return concrete
	case []interface{}:
		for _, value := range concrete {
			unknown := anyUnknown(value)
			if unknown {
				return true
			}
		}
	case map[string]interface{}:
		for _, value := range concrete {
			unknown := anyUnknown(value)
			if unknown {
				return true
			}
		}
	}
	return false
}
