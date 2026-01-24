package javascript

import (
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

type httpClient struct {
	baseURL          *url.URL
	headers          map[string]string
	timeout          time.Duration
	throwOnHTTPError bool
	vm               *goja.Runtime
}

func registerHttpModule(registry *require.Registry) {
	registry.RegisterNativeModule("gopherbot_http", func(rt *goja.Runtime, module *goja.Object) {
		exports := module.Get("exports").(*goja.Object)
		clientFactory := func(call goja.FunctionCall) goja.Value {
			client := newHTTPClient(rt, call)
			obj := rt.NewObject()
			_ = obj.Set("request", client.request)
			_ = obj.Set("getJSON", client.getJSON)
			_ = obj.Set("postJSON", client.postJSON)
			return obj
		}
		_ = exports.Set("createClient", clientFactory)
	})
}

func newHTTPClient(vm *goja.Runtime, call goja.FunctionCall) *httpClient {
	var opts map[string]interface{}
	if len(call.Arguments) > 0 && !isUndefinedOrNull(call.Arguments[0]) {
		raw := call.Arguments[0].Export()
		var ok bool
		opts, ok = raw.(map[string]interface{})
		if !ok {
			panic(vm.NewTypeError("createClient: options must be an object"))
		}
	}

	baseURL, err := parseBaseURL(opts)
	if err != nil {
		panic(vm.NewTypeError(err.Error()))
	}
	headers, err := parseHeaders(opts, "headers")
	if err != nil {
		panic(vm.NewTypeError(err.Error()))
	}
	timeout, err := parseTimeoutMs(opts, "timeoutMs")
	if err != nil {
		panic(vm.NewTypeError(err.Error()))
	}
	throwOnHTTPError, err := parseBool(opts, "throwOnHTTPError", false)
	if err != nil {
		panic(vm.NewTypeError(err.Error()))
	}

	return &httpClient{
		baseURL:          baseURL,
		headers:          headers,
		timeout:          timeout,
		throwOnHTTPError: throwOnHTTPError,
		vm:               vm,
	}
}

func (c *httpClient) request(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) == 0 {
		panic(c.vm.NewTypeError("request: options are required"))
	}
	opts, err := parseRequestOptions(c.vm, call.Arguments[0])
	if err != nil {
		panic(c.vm.NewTypeError(err.Error()))
	}

	respVal, err := c.doRequest(opts)
	if err != nil {
		panic(c.jsError(err))
	}
	return respVal
}

func (c *httpClient) getJSON(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) == 0 {
		panic(c.vm.NewTypeError("getJSON: path is required"))
	}
	path := call.Arguments[0].String()
	var opts map[string]interface{}
	if len(call.Arguments) > 1 && !isUndefinedOrNull(call.Arguments[1]) {
		raw := call.Arguments[1].Export()
		var ok bool
		opts, ok = raw.(map[string]interface{})
		if !ok {
			panic(c.vm.NewTypeError("getJSON: options must be an object"))
		}
	}

	req := requestOptions{
		Method: "GET",
		Path:   path,
		Opts:   opts,
	}

	bodyVal, err := c.doRequestJSON(req)
	if err != nil {
		panic(c.jsError(err))
	}
	return bodyVal
}

func (c *httpClient) postJSON(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(c.vm.NewTypeError("postJSON: path and payload are required"))
	}
	path := call.Arguments[0].String()
	payload := call.Arguments[1].Export()
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		panic(c.vm.NewTypeError(fmt.Sprintf("postJSON: unable to serialize payload: %s", err)))
	}
	var opts map[string]interface{}
	if len(call.Arguments) > 2 && !isUndefinedOrNull(call.Arguments[2]) {
		raw := call.Arguments[2].Export()
		var ok bool
		opts, ok = raw.(map[string]interface{})
		if !ok {
			panic(c.vm.NewTypeError("postJSON: options must be an object"))
		}
	}

	req := requestOptions{
		Method: "POST",
		Path:   path,
		Body:   bodyBytes,
		Opts:   opts,
	}

	bodyVal, err := c.doRequestJSON(req)
	if err != nil {
		panic(c.jsError(err))
	}
	return bodyVal
}

func (c *httpClient) doRequestJSON(req requestOptions) (goja.Value, error) {
	headers, err := parseHeaders(req.Opts, "headers")
	if err != nil {
		return nil, err
	}
	if headers == nil {
		headers = make(map[string]string)
	}
	if _, ok := headers["Accept"]; !ok {
		headers["Accept"] = "application/json"
	}
	if req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH" {
		if _, ok := headers["Content-Type"]; !ok {
			headers["Content-Type"] = "application/json"
		}
	}
	req.Headers = headers

	timeout, err := parseTimeoutMs(req.Opts, "timeoutMs")
	if err != nil {
		return nil, err
	}
	req.Timeout = timeout
	req.TimeoutSet = hasKey(req.Opts, "timeoutMs")

	throwOnHTTPError, err := parseBool(req.Opts, "throwOnHTTPError", false)
	if err != nil {
		return nil, err
	}
	req.ThrowOnHTTPError = throwOnHTTPError
	req.ThrowOnHTTPErrorSet = hasKey(req.Opts, "throwOnHTTPError")

	query, err := parseQuery(req.Opts, "query")
	if err != nil {
		return nil, err
	}
	req.Query = query

	respVal, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	respObj := respVal.ToObject(c.vm)
	body := respObj.Get("body").String()
	return c.parseJSON(body)
}

func (c *httpClient) doRequest(opts requestOptions) (goja.Value, error) {
	method := strings.ToUpper(strings.TrimSpace(opts.Method))
	if method == "" {
		method = "GET"
	}

	fullURL, err := c.buildURL(opts)
	if err != nil {
		return nil, err
	}

	bodyReader, err := bodyReaderFromValue(opts.Body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, err
	}

	headers := mergeHeaders(c.headers, opts.Headers)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	timeout := c.timeout
	if opts.TimeoutSet {
		timeout = opts.Timeout
	}

	client := &http.Client{}
	if timeout > 0 {
		client.Timeout = timeout
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	body := string(respBody)
	statusText := statusTextFromResponse(resp)
	throwOnHTTPError := c.throwOnHTTPError
	if opts.ThrowOnHTTPErrorSet {
		throwOnHTTPError = opts.ThrowOnHTTPError
	}
	if throwOnHTTPError && resp.StatusCode >= 400 {
		return nil, &httpStatusError{
			Status:     resp.StatusCode,
			StatusText: statusText,
			Body:       body,
		}
	}

	responseObj := c.vm.NewObject()
	_ = responseObj.Set("status", resp.StatusCode)
	_ = responseObj.Set("statusText", statusText)
	_ = responseObj.Set("headers", resp.Header)
	_ = responseObj.Set("body", body)
	_ = responseObj.Set("json", func(call goja.FunctionCall) goja.Value {
		value, err := c.parseJSON(body)
		if err != nil {
			panic(c.vm.NewTypeError(err.Error()))
		}
		return value
	})

	return responseObj, nil
}

func (c *httpClient) buildURL(opts requestOptions) (string, error) {
	var rawURL string
	if opts.URL != "" {
		rawURL = opts.URL
	} else if opts.Path != "" {
		rawURL = opts.Path
	}
	if rawURL == "" {
		return "", fmt.Errorf("request: path or url is required")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	if parsed.Scheme == "" {
		if c.baseURL == nil {
			return "", fmt.Errorf("request: baseURL is required when path is relative")
		}
		parsed, err = c.baseURL.Parse(rawURL)
		if err != nil {
			return "", err
		}
	}

	if len(opts.Query) > 0 {
		values := parsed.Query()
		for key, list := range opts.Query {
			for _, value := range list {
				values.Add(key, value)
			}
		}
		parsed.RawQuery = values.Encode()
	}

	return parsed.String(), nil
}

func (c *httpClient) parseJSON(body string) (goja.Value, error) {
	var parsed interface{}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}
	return c.vm.ToValue(parsed), nil
}

func (c *httpClient) jsError(err error) goja.Value {
	if httpErr, ok := err.(*httpStatusError); ok {
		errObj := c.vm.NewTypeError(fmt.Sprintf("HTTP %d %s", httpErr.Status, httpErr.StatusText))
		obj := errObj.ToObject(c.vm)
		_ = obj.Set("status", httpErr.Status)
		_ = obj.Set("statusText", httpErr.StatusText)
		_ = obj.Set("body", httpErr.Body)
		return errObj
	}
	return c.vm.NewGoError(err)
}

type httpStatusError struct {
	Status     int
	StatusText string
	Body       string
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("HTTP %d %s", e.Status, e.StatusText)
}

type requestOptions struct {
	Method              string
	Path                string
	URL                 string
	Query               map[string][]string
	Headers             map[string]string
	Body                interface{}
	Timeout             time.Duration
	TimeoutSet          bool
	ThrowOnHTTPError    bool
	ThrowOnHTTPErrorSet bool
	Opts                map[string]interface{}
}

func parseRequestOptions(vm *goja.Runtime, val goja.Value) (requestOptions, error) {
	if isUndefinedOrNull(val) {
		return requestOptions{}, fmt.Errorf("request: options are required")
	}
	raw := val.Export()
	opts, ok := raw.(map[string]interface{})
	if !ok {
		return requestOptions{}, fmt.Errorf("request: options must be an object")
	}

	method := parseString(opts, "method")
	path := parseString(opts, "path")
	rawURL := parseString(opts, "url")
	headers, err := parseHeaders(opts, "headers")
	if err != nil {
		return requestOptions{}, err
	}
	query, err := parseQuery(opts, "query")
	if err != nil {
		return requestOptions{}, err
	}
	timeout, err := parseTimeoutMs(opts, "timeoutMs")
	if err != nil {
		return requestOptions{}, err
	}
	timeoutSet := hasKey(opts, "timeoutMs")
	throwOnHTTPError, err := parseBool(opts, "throwOnHTTPError", false)
	if err != nil {
		return requestOptions{}, err
	}
	throwOnHTTPErrorSet := hasKey(opts, "throwOnHTTPError")

	body, err := parseBody(opts)
	if err != nil {
		return requestOptions{}, err
	}

	return requestOptions{
		Method:              method,
		Path:                path,
		URL:                 rawURL,
		Query:               query,
		Headers:             headers,
		Body:                body,
		Timeout:             timeout,
		TimeoutSet:          timeoutSet,
		ThrowOnHTTPError:    throwOnHTTPError,
		ThrowOnHTTPErrorSet: throwOnHTTPErrorSet,
		Opts:                opts,
	}, nil
}

func parseBaseURL(opts map[string]interface{}) (*url.URL, error) {
	if opts == nil {
		return nil, nil
	}
	raw, ok := opts["baseURL"]
	if !ok {
		return nil, nil
	}
	str, ok := raw.(string)
	if !ok || strings.TrimSpace(str) == "" {
		return nil, fmt.Errorf("createClient: baseURL must be a non-empty string")
	}
	parsed, err := url.Parse(str)
	if err != nil {
		return nil, fmt.Errorf("createClient: invalid baseURL: %w", err)
	}
	if parsed.Scheme == "" {
		return nil, fmt.Errorf("createClient: baseURL must include a scheme")
	}
	return parsed, nil
}

func parseHeaders(opts map[string]interface{}, key string) (map[string]string, error) {
	if opts == nil {
		return nil, nil
	}
	raw, ok := opts[key]
	if !ok || raw == nil {
		return nil, nil
	}
	headerMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%s: headers must be an object", key)
	}
	out := make(map[string]string, len(headerMap))
	for k, v := range headerMap {
		out[k] = fmt.Sprint(v)
	}
	return out, nil
}

func parseQuery(opts map[string]interface{}, key string) (map[string][]string, error) {
	if opts == nil {
		return nil, nil
	}
	raw, ok := opts[key]
	if !ok || raw == nil {
		return nil, nil
	}
	queryMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("query must be an object")
	}
	out := make(map[string][]string, len(queryMap))
	for k, v := range queryMap {
		switch val := v.(type) {
		case []interface{}:
			values := make([]string, 0, len(val))
			for _, item := range val {
				values = append(values, fmt.Sprint(item))
			}
			out[k] = values
		default:
			out[k] = []string{fmt.Sprint(val)}
		}
	}
	return out, nil
}

func parseTimeoutMs(opts map[string]interface{}, key string) (time.Duration, error) {
	if opts == nil {
		return 0, nil
	}
	raw, ok := opts[key]
	if !ok {
		return 0, nil
	}
	switch val := raw.(type) {
	case int64:
		if val <= 0 {
			return 0, nil
		}
		return time.Duration(val) * time.Millisecond, nil
	case float64:
		if val <= 0 {
			return 0, nil
		}
		return time.Duration(val) * time.Millisecond, nil
	case int:
		if val <= 0 {
			return 0, nil
		}
		return time.Duration(val) * time.Millisecond, nil
	default:
		return 0, fmt.Errorf("%s must be a number (milliseconds)", key)
	}
}

func parseBool(opts map[string]interface{}, key string, fallback bool) (bool, error) {
	if opts == nil {
		return fallback, nil
	}
	raw, ok := opts[key]
	if !ok {
		return fallback, nil
	}
	val, ok := raw.(bool)
	if !ok {
		return fallback, fmt.Errorf("%s must be a boolean", key)
	}
	return val, nil
}

func parseString(opts map[string]interface{}, key string) string {
	if opts == nil {
		return ""
	}
	raw, ok := opts[key]
	if !ok || raw == nil {
		return ""
	}
	str, ok := raw.(string)
	if !ok {
		return ""
	}
	return str
}

func parseBody(opts map[string]interface{}) (interface{}, error) {
	if opts == nil {
		return nil, nil
	}
	raw, ok := opts["body"]
	if !ok || raw == nil {
		return nil, nil
	}
	switch val := raw.(type) {
	case string:
		return val, nil
	case []byte:
		return val, nil
	default:
		return nil, fmt.Errorf("request: body must be a string or byte array (use postJSON for JSON)")
	}
}

func bodyReaderFromValue(body interface{}) (io.Reader, error) {
	switch val := body.(type) {
	case nil:
		return nil, nil
	case string:
		return strings.NewReader(val), nil
	case []byte:
		return strings.NewReader(string(val)), nil
	default:
		return nil, fmt.Errorf("request: body must be a string or byte array")
	}
}

func mergeHeaders(clientHeaders, requestHeaders map[string]string) map[string]string {
	if len(clientHeaders) == 0 && len(requestHeaders) == 0 {
		return nil
	}
	merged := make(map[string]string, len(clientHeaders)+len(requestHeaders))
	for k, v := range clientHeaders {
		merged[k] = v
	}
	for k, v := range requestHeaders {
		merged[k] = v
	}
	return merged
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

func hasKey(opts map[string]interface{}, key string) bool {
	if opts == nil {
		return false
	}
	_, ok := opts[key]
	return ok
}

func isUndefinedOrNull(val goja.Value) bool {
	return val == nil || val == goja.Undefined() || val == goja.Null()
}
