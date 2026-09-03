package javascript

import (
	"reflect"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func TestParseGoValueToJSCreatesNativeNestedContainers(t *testing.T) {
	runtime := goja.New()
	value, err := parseGoValueToJS(runtime, map[string]interface{}{
		"counter": float64(1),
		"items": []interface{}{
			map[string]interface{}{"id": "one"},
		},
		"nested": map[string]interface{}{
			"values": []interface{}{"a"},
		},
	})
	if err != nil {
		t.Fatalf("parseGoValueToJS() error = %v", err)
	}
	if err := runtime.Set("datum", value); err != nil {
		t.Fatalf("runtime.Set() error = %v", err)
	}

	result, err := runtime.RunString(`
		if (datum.items !== datum.items) {
			throw new Error("nested array identity was not preserved");
		}
		datum.items.push({id: "two"});
		datum.items.splice(0, 1);
		datum.nested.values.push("b");
		datum.counter += 1;
		datum;
	`)
	if err != nil {
		t.Fatalf("mutating converted value: %v", err)
	}

	converted, err := parseJSValueToGo(result)
	if err != nil {
		t.Fatalf("parseJSValueToGo() error = %v", err)
	}
	want := map[string]interface{}{
		"counter": int64(2),
		"items": []interface{}{
			map[string]interface{}{"id": "two"},
		},
		"nested": map[string]interface{}{
			"values": []interface{}{"a", "b"},
		},
	}
	if !reflect.DeepEqual(converted, want) {
		t.Fatalf("converted datum = %#v, want %#v", converted, want)
	}
}

func TestParseGoValueToJSPreservesProtoDataProperty(t *testing.T) {
	runtime := goja.New()
	value, err := parseGoValueToJS(runtime, map[string]interface{}{
		"__proto__": map[string]interface{}{"polluted": true},
	})
	if err != nil {
		t.Fatalf("parseGoValueToJS() error = %v", err)
	}
	if err := runtime.Set("datum", value); err != nil {
		t.Fatalf("runtime.Set() error = %v", err)
	}

	result, err := runtime.RunString(`
		Object.prototype.hasOwnProperty.call(datum, "__proto__") &&
			datum.__proto__.polluted === true &&
			Object.getPrototypeOf(datum) === Object.prototype;
	`)
	if err != nil {
		t.Fatalf("checking __proto__ property: %v", err)
	}
	if valid, ok := result.Export().(bool); !ok || !valid {
		t.Fatalf("__proto__ was not preserved as an own data property")
	}
}

func TestParseJSValueToGoAcceptsStrictJSONData(t *testing.T) {
	runtime := goja.New()
	value, err := runtime.RunString(`({
		name: "example",
		enabled: true,
		count: 2,
		ratio: 1.5,
		nothing: null,
		items: ["one", {id: 2}]
	})`)
	if err != nil {
		t.Fatalf("creating JavaScript value: %v", err)
	}

	converted, err := parseJSValueToGo(value)
	if err != nil {
		t.Fatalf("parseJSValueToGo() error = %v", err)
	}
	want := map[string]interface{}{
		"name":    "example",
		"enabled": true,
		"count":   int64(2),
		"ratio":   float64(1.5),
		"nothing": nil,
		"items": []interface{}{
			"one",
			map[string]interface{}{"id": int64(2)},
		},
	}
	if !reflect.DeepEqual(converted, want) {
		t.Fatalf("converted datum = %#v, want %#v", converted, want)
	}
}

func TestParseJSValueToGoRejectsUnsupportedData(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		wantError  string
		forbidText string
	}{
		{name: "undefined", source: `({bad: undefined})`, wantError: `unsupported undefined value at $["bad"]`},
		{name: "function", source: `({bad: function() {}})`, wantError: `unsupported function at $["bad"]`},
		{name: "symbol", source: `({bad: Symbol("value")})`, wantError: `unsupported symbol at $["bad"]`},
		{name: "bigint", source: `({bad: 1n})`, wantError: `unsupported BigInt at $["bad"]`},
		{name: "nan", source: `({bad: NaN})`, wantError: `unsupported non-finite number at $["bad"]`},
		{name: "infinity", source: `({bad: Infinity})`, wantError: `unsupported non-finite number at $["bad"]`},
		{name: "cycle", source: `(function() { const value = {}; value.self = value; return value; })()`, wantError: `cycle at $["self"] references $`},
		{name: "sparse array", source: `[1, , 3]`, wantError: `unsupported sparse array at $`},
		{name: "array property", source: `(function() { const value = [1]; value.note = "x"; return value; })()`, wantError: `unsupported non-index array property "note" at $`},
		{name: "date", source: `new Date(0)`, wantError: `unsupported JavaScript object type Date at $`},
		{name: "non-enumerable property", source: `(function() { const value = {}; Object.defineProperty(value, "hidden", {value: 1}); return value; })()`, wantError: `unsupported non-enumerable property at $`},
		{name: "throwing getter", source: `({get bad() { throw new Error("super-secret"); }})`, wantError: `JavaScript exception while reading $["bad"]`, forbidText: "super-secret"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runtime := goja.New()
			value, err := runtime.RunString(test.source)
			if err != nil {
				t.Fatalf("creating JavaScript value: %v", err)
			}
			_, err = parseJSValueToGo(value)
			if err == nil {
				t.Fatal("parseJSValueToGo() error = nil")
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("parseJSValueToGo() error = %q, want substring %q", err, test.wantError)
			}
			if test.forbidText != "" && strings.Contains(err.Error(), test.forbidText) {
				t.Fatalf("parseJSValueToGo() error leaked rejected value: %q", err)
			}
		})
	}
}

func TestParseJSValueToGoAllowsRepeatedReferences(t *testing.T) {
	runtime := goja.New()
	value, err := runtime.RunString(`(function() {
		const child = {value: 1};
		return {first: child, second: child};
	})()`)
	if err != nil {
		t.Fatalf("creating JavaScript value: %v", err)
	}

	converted, err := parseJSValueToGo(value)
	if err != nil {
		t.Fatalf("parseJSValueToGo() error = %v", err)
	}
	wantChild := map[string]interface{}{"value": int64(1)}
	want := map[string]interface{}{"first": wantChild, "second": wantChild}
	if !reflect.DeepEqual(converted, want) {
		t.Fatalf("converted datum = %#v, want %#v", converted, want)
	}
}
