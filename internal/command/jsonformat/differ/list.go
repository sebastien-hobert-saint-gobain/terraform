package differ

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/list"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

func (v Value) computeAttributeChangeAsList(elementType cty.Type) change.Change {
	var elements []change.Change
	current := v.getDefaultActionForIteration()

	sliceValue := v.asSlice()

	list.Process(sliceValue.Before, sliceValue.After, elementType.IsObjectType(), func(beforeIx, afterIx int) {
		value := sliceValue.getChild(beforeIx, afterIx, false)
		element := value.ComputeChange(elementType)
		elements = append(elements, element)
		current = compareActions(current, element.GetAction())
	})
	return change.New(change.List(elements), current, v.replacePath())
}

func (v Value) computeAttributeChangeAsNestedList(attributes map[string]*jsonprovider.Attribute) change.Change {
	var elements []change.Change
	current := v.getDefaultActionForIteration()
	v.processNestedList(func(value Value) {
		element := value.ComputeChange(attributes)
		elements = append(elements, element)
		current = compareActions(current, element.GetAction())
	})
	return change.New(change.NestedList(elements), current, v.replacePath())
}

func (v Value) computeBlockChangesAsList(block *jsonprovider.Block) ([]change.Change, plans.Action) {
	var elements []change.Change
	current := v.getDefaultActionForIteration()
	v.processNestedList(func(value Value) {
		element := value.ComputeChange(block)
		elements = append(elements, element)
		current = compareActions(current, element.GetAction())
	})
	return elements, current
}

func (v Value) processNestedList(process func(value Value)) {
	sliceValue := v.asSlice()
	for ix := 0; ix < len(sliceValue.Before) || ix < len(sliceValue.After); ix++ {
		process(sliceValue.getChild(ix, ix, false))
	}
}
