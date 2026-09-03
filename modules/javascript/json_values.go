package javascript

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"

	"github.com/dop251/goja"
)

// parseGoValueToJS converts JSON-shaped Go data into native JavaScript objects
// and arrays. Passing maps and slices directly to Runtime.ToValue exposes Goja
// host objects whose mutation semantics differ from ordinary JavaScript values.
func parseGoValueToJS(rt *goja.Runtime, data interface{}) (goja.Value, error) {
	return nativeJSValue(rt, data)
}

func nativeJSValue(rt *goja.Runtime, data interface{}) (goja.Value, error) {
	switch value := data.(type) {
	case nil:
		return goja.Null(), nil
	case bool, string, int64:
		return rt.ToValue(value), nil
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("non-finite Go number is not JSON-compatible")
		}
		return rt.ToValue(value), nil
	case []interface{}:
		items := make([]interface{}, len(value))
		for index, item := range value {
			converted, err := nativeJSValue(rt, item)
			if err != nil {
				return nil, fmt.Errorf("array index %d: %w", index, err)
			}
			items[index] = converted
		}
		return rt.NewArray(items...), nil
	case map[string]interface{}:
		object := rt.NewObject()
		for key, item := range value {
			converted, err := nativeJSValue(rt, item)
			if err != nil {
				return nil, fmt.Errorf("object property %q: %w", key, err)
			}
			if err := object.DefineDataProperty(key, converted, goja.FLAG_TRUE, goja.FLAG_TRUE, goja.FLAG_TRUE); err != nil {
				return nil, fmt.Errorf("defining object property %q: %w", key, err)
			}
		}
		return object, nil
	default:
		// Robot API implementations normally return generic JSON values. Keep the
		// boundary robust for typed maps, slices, and structs without exposing the
		// resulting Go value as a host object.
		encoded, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("encoding Go value as JSON: %w", err)
		}
		var normalized interface{}
		if err := json.Unmarshal(encoded, &normalized); err != nil {
			return nil, fmt.Errorf("decoding normalized JSON value: %w", err)
		}
		return nativeJSValue(rt, normalized)
	}
}

// parseJSValueToGo strictly converts JavaScript data to values accepted by the
// brain's JSON contract. Unlike JSON.stringify, it rejects unsupported values
// instead of silently dropping or coercing them.
func parseJSValueToGo(value goja.Value) (interface{}, error) {
	converter := jsDatumConverter{
		stack: make(map[*goja.Object]string),
	}
	return converter.convert(value, "$")
}

type jsDatumConverter struct {
	stack map[*goja.Object]string
}

func (converter *jsDatumConverter) convert(value goja.Value, path string) (interface{}, error) {
	if value == nil || goja.IsUndefined(value) {
		return nil, fmt.Errorf("unsupported undefined value at %s", path)
	}
	if goja.IsNull(value) {
		return nil, nil
	}
	if _, isSymbol := value.(*goja.Symbol); isSymbol {
		return nil, fmt.Errorf("unsupported symbol at %s", path)
	}
	if _, callable := goja.AssertFunction(value); callable {
		return nil, fmt.Errorf("unsupported function at %s", path)
	}

	if object, isObject := value.(*goja.Object); isObject {
		return converter.convertObject(object, path)
	}

	exported := value.Export()
	switch primitive := exported.(type) {
	case bool:
		return primitive, nil
	case string:
		return primitive, nil
	case int64:
		return primitive, nil
	case float64:
		if math.IsNaN(primitive) || math.IsInf(primitive, 0) {
			return nil, fmt.Errorf("unsupported non-finite number at %s", path)
		}
		return primitive, nil
	case *big.Int:
		return nil, fmt.Errorf("unsupported BigInt at %s", path)
	default:
		return nil, fmt.Errorf("unsupported JavaScript value of type %T at %s", exported, path)
	}
}

func (converter *jsDatumConverter) convertObject(object *goja.Object, path string) (interface{}, error) {
	if previousPath, exists := converter.stack[object]; exists {
		return nil, fmt.Errorf("cycle at %s references %s", path, previousPath)
	}
	converter.stack[object] = path
	defer delete(converter.stack, object)

	if err := rejectSymbolProperties(object, path); err != nil {
		return nil, err
	}

	switch object.ClassName() {
	case "Array":
		return converter.convertArray(object, path)
	case "Object":
		return converter.convertPlainObject(object, path)
	default:
		return nil, fmt.Errorf("unsupported JavaScript object type %s at %s", object.ClassName(), path)
	}
}

func (converter *jsDatumConverter) convertArray(object *goja.Object, path string) (interface{}, error) {
	lengthValue, err := getJSProperty(object, "length", path)
	if err != nil {
		return nil, err
	}
	length := lengthValue.ToInteger()
	if length < 0 {
		return nil, fmt.Errorf("invalid array length at %s", path)
	}

	propertyNames, err := getJSPropertyNames(object, path)
	if err != nil {
		return nil, err
	}
	indexes := make(map[int64]struct{}, len(propertyNames))
	for _, name := range propertyNames {
		if name == "length" {
			continue
		}
		index, parseErr := strconv.ParseUint(name, 10, 32)
		if parseErr != nil || strconv.FormatUint(index, 10) != name || int64(index) >= length {
			return nil, fmt.Errorf("unsupported non-index array property %q at %s", name, path)
		}
		indexes[int64(index)] = struct{}{}
	}
	if int64(len(indexes)) != length {
		return nil, fmt.Errorf("unsupported sparse array at %s", path)
	}

	converted := make([]interface{}, int(length))
	for index := int64(0); index < length; index++ {
		itemPath := fmt.Sprintf("%s[%d]", path, index)
		item, err := getJSProperty(object, strconv.FormatInt(index, 10), itemPath)
		if err != nil {
			return nil, err
		}
		convertedItem, err := converter.convert(item, itemPath)
		if err != nil {
			return nil, err
		}
		converted[index] = convertedItem
	}
	return converted, nil
}

func (converter *jsDatumConverter) convertPlainObject(object *goja.Object, path string) (interface{}, error) {
	keys, err := getJSKeys(object, path)
	if err != nil {
		return nil, err
	}
	propertyNames, err := getJSPropertyNames(object, path)
	if err != nil {
		return nil, err
	}
	if len(propertyNames) != len(keys) {
		return nil, fmt.Errorf("unsupported non-enumerable property at %s", path)
	}

	converted := make(map[string]interface{}, len(keys))
	for _, key := range keys {
		itemPath := path + "[" + strconv.Quote(key) + "]"
		item, err := getJSProperty(object, key, itemPath)
		if err != nil {
			return nil, err
		}
		convertedItem, err := converter.convert(item, itemPath)
		if err != nil {
			return nil, err
		}
		converted[key] = convertedItem
	}
	return converted, nil
}

func rejectSymbolProperties(object *goja.Object, path string) (err error) {
	defer recoverJSPropertyError(path, &err)
	if len(object.Symbols()) != 0 {
		return fmt.Errorf("unsupported symbol-keyed property at %s", path)
	}
	return nil
}

func getJSKeys(object *goja.Object, path string) (keys []string, err error) {
	defer recoverJSPropertyError(path, &err)
	return object.Keys(), nil
}

func getJSPropertyNames(object *goja.Object, path string) (names []string, err error) {
	defer recoverJSPropertyError(path, &err)
	return object.GetOwnPropertyNames(), nil
}

func getJSProperty(object *goja.Object, name, path string) (value goja.Value, err error) {
	defer recoverJSPropertyError(path, &err)
	return object.Get(name), nil
}

func recoverJSPropertyError(path string, err *error) {
	if recovered := recover(); recovered != nil {
		if _, isJSException := recovered.(*goja.Exception); isJSException {
			*err = fmt.Errorf("JavaScript exception while reading %s", path)
			return
		}
		panic(recovered)
	}
}
