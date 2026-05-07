package javascript

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

type jsHTTPModule struct {
	vm *goja.Runtime
}

type jsHTTPRequestOptions struct {
	Query     interface{}
	Cookies   map[string]string
	Headers   map[string]string
	Body      interface{}
	Form      interface{}
	Timeout   time.Duration
	TimeoutOn bool
	Auth      *jsHTTPAuth
}

type jsHTTPAuth struct {
	User string
	Pass string
}

func registerHttpModule(registry *require.Registry) {
	registry.RegisterNativeModule("http", func(rt *goja.Runtime, module *goja.Object) {
		exports := module.Get("exports").(*goja.Object)
		mod := &jsHTTPModule{vm: rt}

		_ = exports.Set("get", mod.verb("GET"))
		_ = exports.Set("delete", mod.verb("DELETE"))
		_ = exports.Set("head", mod.verb("HEAD"))
		_ = exports.Set("patch", mod.verb("PATCH"))
		_ = exports.Set("post", mod.verb("POST"))
		_ = exports.Set("put", mod.verb("PUT"))
		_ = exports.Set("request", mod.request)
		_ = exports.Set("requestBatch", mod.requestBatch)
		_ = exports.Set("request_batch", mod.requestBatch)
	})
}

func (m *jsHTTPModule) verb(method string) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(m.vm.NewTypeError("%s: url is required", strings.ToLower(method)))
		}
		return m.doRequestValue(method, call.Arguments[0], optionalArg(call, 1))
	}
}

func (m *jsHTTPModule) request(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(m.vm.NewTypeError("request: method and url are required"))
	}
	return m.doRequestValue(call.Arguments[0].String(), call.Arguments[1], optionalArg(call, 2))
}

func (m *jsHTTPModule) requestBatch(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) == 0 {
		panic(m.vm.NewTypeError("requestBatch: requests array is required"))
	}
	requests, ok := call.Arguments[0].Export().([]interface{})
	if !ok {
		panic(m.vm.NewTypeError("requestBatch: requests must be an array"))
	}

	responses := m.vm.NewArray()
	errors := m.vm.NewArray()
	hasErrors := false
	for i, raw := range requests {
		method, rawURL, opts, ok := m.batchRequest(raw)
		if !ok {
			_ = responses.Set(strconv.Itoa(i), goja.Null())
			_ = errors.Set(strconv.Itoa(i), "request must be an object")
			hasErrors = true
			continue
		}
		response, err := m.doRequest(method, rawURL, opts)
		if err != nil {
			_ = responses.Set(strconv.Itoa(i), goja.Null())
			_ = errors.Set(strconv.Itoa(i), err.Error())
			hasErrors = true
			continue
		}
		_ = responses.Set(strconv.Itoa(i), response)
		_ = errors.Set(strconv.Itoa(i), goja.Null())
	}
	if hasErrors {
		out := m.vm.NewArray()
		_ = out.Set("0", responses)
		_ = out.Set("1", errors)
		return out
	}
	return responses
}

func (m *jsHTTPModule) batchRequest(raw interface{}) (string, string, goja.Value, bool) {
	switch request := raw.(type) {
	case map[string]interface{}:
		method, _ := request["method"].(string)
		if method == "" {
			method = "GET"
		}
		rawURL, _ := request["url"].(string)
		return method, rawURL, m.vm.ToValue(request["options"]), rawURL != ""
	case []interface{}:
		if len(request) < 2 {
			return "", "", nil, false
		}
		method := fmt.Sprint(request[0])
		rawURL := fmt.Sprint(request[1])
		opts := goja.Undefined()
		if len(request) > 2 {
			opts = m.vm.ToValue(request[2])
		}
		return method, rawURL, opts, rawURL != ""
	default:
		return "", "", nil, false
	}
}

func (m *jsHTTPModule) doRequestValue(method string, rawURL goja.Value, rawOpts goja.Value) goja.Value {
	response, err := m.doRequest(method, rawURL.String(), rawOpts)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return response
}

func (m *jsHTTPModule) doRequest(method, rawURL string, rawOpts goja.Value) (goja.Value, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "GET"
	}
	if strings.TrimSpace(rawURL) == "" {
		return nil, fmt.Errorf("request: url is required")
	}

	opts, err := parseJSHTTPRequestOptions(rawOpts)
	if err != nil {
		return nil, err
	}

	fullURL, err := buildJSHTTPURL(rawURL, opts.Query)
	if err != nil {
		return nil, err
	}

	body, contentType, err := jsHTTPBodyReader(opts.Body, opts.Form)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, err
	}

	for name, value := range opts.Headers {
		req.Header.Set(name, value)
	}
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}
	for name, value := range opts.Cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	if opts.Auth != nil {
		req.SetBasicAuth(opts.Auth.User, opts.Auth.Pass)
	}
	if opts.TimeoutOn {
		ctx, cancel := context.WithTimeout(req.Context(), opts.Timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return m.responseObject(resp, respBody), nil
}

func (m *jsHTTPModule) responseObject(resp *http.Response, body []byte) goja.Value {
	obj := m.vm.NewObject()
	bodyString := string(body)
	statusText := statusTextFromResponse(resp)
	contentType := resp.Header.Get("Content-Type")

	_ = obj.Set("body", bodyString)
	_ = obj.Set("bodySize", len(body))
	_ = obj.Set("headers", responseHeaders(resp.Header))
	_ = obj.Set("cookies", responseCookies(resp.Cookies()))
	_ = obj.Set("statusCode", resp.StatusCode)
	_ = obj.Set("statusText", statusText)
	_ = obj.Set("ok", resp.StatusCode >= 200 && resp.StatusCode < 400)
	if resp.Request != nil && resp.Request.URL != nil {
		_ = obj.Set("url", resp.Request.URL.String())
	}

	jsonValue := goja.Null()
	if isJSONContentType(contentType) {
		var parsed interface{}
		if err := json.Unmarshal(body, &parsed); err == nil {
			jsonValue = m.vm.ToValue(parsed)
		}
	}
	_ = obj.Set("json", jsonValue)
	return obj
}

func parseJSHTTPRequestOptions(raw goja.Value) (jsHTTPRequestOptions, error) {
	if isUndefinedOrNull(raw) {
		return jsHTTPRequestOptions{}, nil
	}
	exported := raw.Export()
	opts, ok := exported.(map[string]interface{})
	if !ok {
		return jsHTTPRequestOptions{}, fmt.Errorf("request options must be an object")
	}

	headers, err := stringMapOption(opts, "headers")
	if err != nil {
		return jsHTTPRequestOptions{}, err
	}
	cookies, err := stringMapOption(opts, "cookies")
	if err != nil {
		return jsHTTPRequestOptions{}, err
	}
	timeout, timeoutOn, err := timeoutOption(opts)
	if err != nil {
		return jsHTTPRequestOptions{}, err
	}
	auth, err := authOption(opts)
	if err != nil {
		return jsHTTPRequestOptions{}, err
	}

	return jsHTTPRequestOptions{
		Query:     opts["query"],
		Cookies:   cookies,
		Headers:   headers,
		Body:      opts["body"],
		Form:      opts["form"],
		Timeout:   timeout,
		TimeoutOn: timeoutOn,
		Auth:      auth,
	}, nil
}

func buildJSHTTPURL(rawURL string, query interface{}) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if query == nil {
		return parsed.String(), nil
	}
	switch q := query.(type) {
	case string:
		parsed.RawQuery = q
	case map[string]interface{}:
		values := parsed.Query()
		for key, value := range q {
			switch v := value.(type) {
			case []interface{}:
				for _, item := range v {
					values.Add(key, fmt.Sprint(item))
				}
			default:
				values.Add(key, fmt.Sprint(v))
			}
		}
		parsed.RawQuery = values.Encode()
	default:
		return "", fmt.Errorf("query must be a string or object")
	}
	return parsed.String(), nil
}

func jsHTTPBodyReader(body, form interface{}) (io.Reader, string, error) {
	if body == nil {
		if form == nil {
			return nil, "", nil
		}
		return strings.NewReader(fmt.Sprint(form)), "application/x-www-form-urlencoded", nil
	}
	switch value := body.(type) {
	case string:
		return strings.NewReader(value), "", nil
	case []byte:
		return bytes.NewReader(value), "", nil
	case []interface{}:
		bytesValue := make([]byte, len(value))
		for i, item := range value {
			n, ok := numericByte(item)
			if !ok {
				return nil, "", fmt.Errorf("body byte array contains non-byte value at index %d", i)
			}
			bytesValue[i] = n
		}
		return bytes.NewReader(bytesValue), "", nil
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return nil, "", fmt.Errorf("unable to encode request body as JSON: %w", err)
		}
		return bytes.NewReader(data), "application/json", nil
	}
}

func stringMapOption(opts map[string]interface{}, key string) (map[string]string, error) {
	raw, ok := opts[key]
	if !ok || raw == nil {
		return nil, nil
	}
	values, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%s must be an object", key)
	}
	out := make(map[string]string, len(values))
	for name, value := range values {
		out[name] = fmt.Sprint(value)
	}
	return out, nil
}

func timeoutOption(opts map[string]interface{}) (time.Duration, bool, error) {
	if raw, ok := opts["timeoutMs"]; ok && raw != nil {
		ms, ok := numberOption(raw)
		if !ok {
			return 0, false, fmt.Errorf("timeoutMs must be a number")
		}
		if ms <= 0 {
			return 0, false, nil
		}
		return time.Duration(ms) * time.Millisecond, true, nil
	}
	raw, ok := opts["timeout"]
	if !ok || raw == nil {
		return 0, false, nil
	}
	switch value := raw.(type) {
	case string:
		duration, err := time.ParseDuration(value)
		if err != nil {
			return 0, false, err
		}
		return duration, duration > 0, nil
	default:
		seconds, ok := numberOption(value)
		if !ok {
			return 0, false, fmt.Errorf("timeout must be a duration string or number of seconds")
		}
		if seconds <= 0 {
			return 0, false, nil
		}
		return time.Duration(seconds) * time.Second, true, nil
	}
}

func authOption(opts map[string]interface{}) (*jsHTTPAuth, error) {
	raw, ok := opts["auth"]
	if !ok || raw == nil {
		return nil, nil
	}
	authMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("auth must be an object")
	}
	user, userOK := authMap["user"]
	pass, passOK := authMap["pass"]
	if !userOK || !passOK || user == nil || pass == nil {
		return nil, fmt.Errorf("auth must include user and pass")
	}
	return &jsHTTPAuth{User: fmt.Sprint(user), Pass: fmt.Sprint(pass)}, nil
}

func responseHeaders(headers http.Header) map[string]string {
	out := make(map[string]string, len(headers))
	for name, values := range headers {
		out[strings.ToLower(name)] = strings.Join(values, ", ")
	}
	return out
}

func responseCookies(cookies []*http.Cookie) map[string]string {
	out := make(map[string]string, len(cookies))
	for _, cookie := range cookies {
		out[cookie.Name] = cookie.Value
	}
	return out
}

func statusTextFromResponse(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	status := strings.TrimSpace(resp.Status)
	code := strconv.Itoa(resp.StatusCode)
	if strings.HasPrefix(status, code) {
		return strings.TrimSpace(strings.TrimPrefix(status, code))
	}
	return status
}

func isJSONContentType(contentType string) bool {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

func numberOption(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

func numericByte(value interface{}) (byte, bool) {
	n, ok := numberOption(value)
	if !ok || n < 0 || n > 255 {
		return 0, false
	}
	return byte(n), true
}

func optionalArg(call goja.FunctionCall, index int) goja.Value {
	if len(call.Arguments) <= index {
		return goja.Undefined()
	}
	return call.Arguments[index]
}

func isUndefinedOrNull(val goja.Value) bool {
	return val == nil || val == goja.Undefined() || val == goja.Null()
}
