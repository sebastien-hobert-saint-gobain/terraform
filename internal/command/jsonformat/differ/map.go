package differ

import (
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
)

func (v Value) computeAttributeChangeAsMap(elementType cty.Type) change.Change {
	mapValue := v.asMap()
	elements, action := change.ProcessMap(mapValue.Before, mapValue.After, func(key string) change.Change {
		return mapValue.getChild(key).ComputeChange(elementType)
	})
	return change.New(change.Map(elements), action, v.replacePath())
}

func (v Value) computeAttributeChangeAsNestedMap(attributes map[string]*jsonprovider.Attribute) change.Change {
	mapValue := v.asMap()
	elements, action := change.ProcessMap(mapValue.Before, mapValue.After, func(key string) change.Change {
		return mapValue.getChild(key).ComputeChange(attributes)
	})
	return change.New(change.Map(elements), action, false)
}

func (v Value) computeBlockChangesAsMap(block *jsonprovider.Block) (map[string]change.Change, plans.Action) {
	mapValue := v.asMap()
	return change.ProcessMap(mapValue.Before, mapValue.After, func(key string) change.Change {
		return mapValue.getChild(key).ComputeChange(block)
	})
}
