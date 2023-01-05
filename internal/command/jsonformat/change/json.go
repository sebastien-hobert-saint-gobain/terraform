package change

import (
	"fmt"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/zclconf/go-cty/cty"
	"reflect"
)

const (
	JsonNumber = "number"
	JsonObject = "object"
	JsonArray  = "array"
	JsonBool   = "bool"
	JsonString = "string"
	JsonNull   = "null"
)

func ComputeChangeForJson(before, after interface{}) Change {
	beforeType := GetJsonType(before)
	afterType := GetJsonType(after)

	if beforeType == afterType || (beforeType == JsonNull || afterType == JsonNull) {
		targetType := beforeType
		if targetType == JsonNull {
			targetType = afterType
		}
		return ComputeJsonUpdateChange(before, after, targetType)
	}

	beforeChange := ComputeJsonUpdateChange(before, nil, beforeType)
	afterChange := ComputeJsonUpdateChange(nil, after, afterType)

	return New(TypeChange(beforeChange, afterChange), plans.Update, false)
}

func ComputeJsonUpdateChange(before, after interface{}, jsonType string) Change {
	switch jsonType {
	case JsonNull:
		return ComputeJsonChangeAsPrimitive(before, after, cty.NilType)
	case JsonBool:
		return ComputeJsonChangeAsPrimitive(before, after, cty.Bool)
	case JsonString:
		return ComputeJsonChangeAsPrimitive(before, after, cty.String)
	case JsonNumber:
		return ComputeJsonChangeAsPrimitive(before, after, cty.Number)
	case JsonObject:
		var b, a map[string]interface{}

		if before != nil {
			b = before.(map[string]interface{})
		}

		if after != nil {
			a = after.(map[string]interface{})
		}

		return ComputeJsonChangeAsObject(b, a)
	case JsonArray:
		var b, a []interface{}

		if before == nil {
			b = nil
		} else {
			b = before.([]interface{})
		}

		if after == nil {
			a = nil
		} else {
			a = after.([]interface{})
		}

		return ComputeJsonChangeAsArray(b, a)
	default:
		panic("unrecognized json type: " + jsonType)
	}
}

func ComputeJsonChangeAsPrimitive(before, after interface{}, ctyType cty.Type) Change {

	var action plans.Action
	switch {
	case before == nil && after != nil:
		action = plans.Create
	case before != nil && after == nil:
		action = plans.Delete
	case reflect.DeepEqual(before, after):
		action = plans.NoOp
	default:
		action = plans.Update
	}

	return New(Primitive(before, after, ctyType), action, false)
}

func ComputeJsonChangeAsObject(before, after map[string]interface{}) Change {
	elements, action := ProcessMap(before, after, func(key string) Change {
		return ComputeChangeForJson(before[key], after[key])
	})
	return New(Object(elements), action, false)
}

func ComputeJsonChangeAsArray(before, after []interface{}) Change {
	elements, action := ProcessList(before, after, func(item interface{}) bool {
		return GetJsonType(item) == JsonObject
	}, func(beforeIx, afterIx int) Change {
		var b, a interface{}

		if beforeIx >= 0 && beforeIx < len(before) {
			b = before[beforeIx]
		}

		if afterIx >= 0 && afterIx < len(after) {
			a = after[afterIx]
		}

		return ComputeChangeForJson(b, a)
	})
	return New(List(elements), action, false)
}

func GetJsonType(json interface{}) string {
	switch json.(type) {
	case []interface{}:
		return JsonArray
	case float64:
		return JsonNumber
	case string:
		return JsonString
	case bool:
		return JsonBool
	case nil:
		return JsonNull
	case map[string]interface{}:
		return JsonObject
	default:
		panic(fmt.Sprintf("unrecognized json type %T", json))
	}
}
