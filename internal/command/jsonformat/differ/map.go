package differ

import (
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
)

func (v Value) computeAttributeChangeAsMap(elementType cty.Type) change.Change {
	current := v.getDefaultActionForIteration()
	elements := make(map[string]change.Change)
	v.processMap(false, func(key string, value Value) {
		element := value.ComputeChange(elementType)
		elements[key] = element
		current = compareActions(current, element.GetAction())
	})
	return change.New(change.Map(elements), current, v.replacePath())
}

func (v Value) computeAttributeChangeAsNestedMap(attributes map[string]*jsonprovider.Attribute) change.Change {
	current := v.getDefaultActionForIteration()
	elements := make(map[string]change.Change)
	v.processMap(true, func(key string, value Value) {
		element := value.ComputeChange(attributes)
		elements[key] = element
		current = compareActions(current, element.GetAction())
	})
	return change.New(change.Map(elements), current, false)
}

func (v Value) computeBlockChangesAsMap(block *jsonprovider.Block) (map[string]change.Change, plans.Action) {
	current := v.getDefaultActionForIteration()
	elements := make(map[string]change.Change)
	v.processMap(false, func(key string, value Value) {
		element := value.ComputeChange(block)
		elements[key] = element
		current = compareActions(current, element.GetAction())
	})
	return elements, current
}

func (v Value) processMap(propagateReplace bool, process func(key string, value Value)) {
	mapValue := v.asMap()

	handled := make(map[string]bool)
	for key := range mapValue.Before {
		handled[key] = true
		process(key, mapValue.getChild(key, propagateReplace))
	}
	for key := range mapValue.After {
		if _, ok := handled[key]; ok {
			continue
		}
		process(key, mapValue.getChild(key, propagateReplace))
	}
}
