package change

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/list"
	"github.com/hashicorp/terraform/internal/plans"
)

type ProcessKey func(key string) Change
type ProcessIndices func(beforeIx, afterIx int) Change

func ProcessMap(before, after map[string]interface{}, process ProcessKey) (map[string]Change, plans.Action) {
	current := plans.NoOp

	if before != nil && after == nil {
		current = plans.Delete
	}

	if before == nil && after != nil {
		current = plans.Create
	}

	elements := make(map[string]Change)

	for key := range before {
		elements[key] = process(key)
		current = compareActions(current, elements[key].GetAction())
	}
	for key := range after {
		if _, ok := elements[key]; ok {
			continue
		}
		elements[key] = process(key)
		current = compareActions(current, elements[key].GetAction())
	}
	return elements, current
}

func ProcessList(before, after []interface{}, isObjType func(item interface{}) bool, process ProcessIndices) ([]Change, plans.Action) {
	current := plans.NoOp

	if before != nil && after == nil {
		current = plans.Delete
	}

	if before == nil && after != nil {
		current = plans.Create
	}

	var elements []Change

	list.Process(before, after, isObjType, func(beforeIx, afterIx int) {
		element := process(beforeIx, afterIx)
		elements = append(elements, element)
		current = compareActions(current, element.GetAction())
	})

	return elements, current
}

func compareActions(current, next plans.Action) plans.Action {
	if next == plans.NoOp {
		return current
	}

	if current != next {
		return plans.Update
	}
	return current
}
