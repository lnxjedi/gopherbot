package lua

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	glua "github.com/yuin/gopher-lua"
)

const httpModuleName = "gopherbot_http"
const httpClientTypeName = "gopherbot_http_client"

type httpClient struct {
	baseURL          *url.URL
	headers          map[string]string
	timeout          time.Duration
	throwOnHTTPError bool
}

func registerHttpModule(L *glua.LState) {
	L.PreloadModule(httpModuleName, func(L *glua.LState) int {
		mod := L.NewTable()
		L.SetField(mod, "create_client", L.NewFunction(func(L *glua.LState) int {
			client, err := newHTTPClient(L, L.OptTable(1, nil))
			if err != nil {
				return pushLuaError(L, err)
			}
			ud := L.NewUserData()
			ud.Value = client
			L.SetMetatable(ud, L.GetTypeMetatable(httpClientTypeName))
			L.Push(ud)
			return 1
		}))
		L.Push(mod)
		return 1
	})

	mt := L.NewTypeMetatable(httpClientTypeName)
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]glua.LGFunction{
		"request":   httpClientRequest,
		"get_json":  httpClientGetJSON,
		"post_json": httpClientPostJSON,
		"put_json":  httpClientPutJSON,
	}))
}

func newHTTPClient(L *glua.LState, tbl *glua.LTable) (*httpClient, error) {
	var opts *glua.LTable
	if tbl != nil {
		opts = tbl
	}
	var baseURL *url.URL
	if opts != nil {
		base := getStringField(opts, "base_url")
		if base != "" {
			parsed, err := url.Parse(base)
			if err != nil || parsed.Scheme == "" {
				return nil, fmt.Errorf("create_client: invalid base_url")
			}
			baseURL = parsed
		}
	}
	headers := parseHeaders(opts, "headers")
	timeout := time.Duration(getNumberField(opts, "timeout_ms")) * time.Millisecond
	throwOnHTTPError := getBoolField(opts, "throw_on_http_error")

	return &httpClient{
		baseURL:          baseURL,
		headers:          headers,
		timeout:          timeout,
		throwOnHTTPError: throwOnHTTPError,
	}, nil
}

func httpClientRequest(L *glua.LState) int {
	client := checkHTTPClient(L)
	opts := L.CheckTable(2)
	resp, err := client.doRequest(L, opts)
	if err != nil {
		return pushLuaError(L, err)
	}
	L.Push(resp)
	L.Push(glua.LNil)
	return 2
}

func httpClientGetJSON(L *glua.LState) int {
	client := checkHTTPClient(L)
	path := L.CheckString(2)
	opts := L.OptTable(3, nil)
	resp, err := client.doRequestJSON(L, "GET", path, glua.LNil, opts)
	if err != nil {
		return pushLuaError(L, err)
	}
	L.Push(resp)
	L.Push(glua.LNil)
	return 2
}

func httpClientPostJSON(L *glua.LState) int {
	client := checkHTTPClient(L)
	path := L.CheckString(2)
	payload := L.Get(3)
	opts := L.OptTable(4, nil)
	resp, err := client.doRequestJSON(L, "POST", path, payload, opts)
	if err != nil {
		return pushLuaError(L, err)
	}
	L.Push(resp)
	L.Push(glua.LNil)
	return 2
}

func httpClientPutJSON(L *glua.LState) int {
	client := checkHTTPClient(L)
	path := L.CheckString(2)
	payload := L.Get(3)
	opts := L.OptTable(4, nil)
	resp, err := client.doRequestJSON(L, "PUT", path, payload, opts)
	if err != nil {
		return pushLuaError(L, err)
	}
	L.Push(resp)
	L.Push(glua.LNil)
	return 2
}

func (c *httpClient) doRequestJSON(L *glua.LState, method, path string, payload glua.LValue, opts *glua.LTable) (glua.LValue, error) {
	headers := parseHeaders(opts, "headers")
	if headers == nil {
		headers = make(map[string]string)
	}
	if _, ok := headers["Accept"]; !ok {
		headers["Accept"] = "application/json"
	}
	if method == "POST" || method == "PUT" || method == "PATCH" {
		if _, ok := headers["Content-Type"]; !ok {
			headers["Content-Type"] = "application/json"
		}
	}

	var body io.Reader
	if payload != glua.LNil && payload != nil && payload.Type() != glua.LTNil {
		converted, err := parseLuaValueToGo(payload, map[*glua.LTable]bool{})
		if err != nil {
			return glua.LNil, err
		}
		data, err := json.Marshal(converted)
		if err != nil {
			return glua.LNil, err
		}
		body = bytes.NewReader(data)
	}

	reqOpts := L.NewTable()
	reqOpts.RawSetString("method", glua.LString(method))
	reqOpts.RawSetString("path", glua.LString(path))
	reqOpts.RawSetString("headers", headersMapToLuaTable(L, headers))
	if opts != nil {
		if val := opts.RawGetString("timeout_ms"); val != glua.LNil {
			reqOpts.RawSetString("timeout_ms", val)
		}
		if val := opts.RawGetString("throw_on_http_error"); val != glua.LNil {
			reqOpts.RawSetString("throw_on_http_error", val)
		}
		if val := opts.RawGetString("query"); val != glua.LNil {
			reqOpts.RawSetString("query", val)
		}
	}
	resp, err := c.doRequestWithBody(L, reqOpts, body)
	if err != nil {
		return glua.LNil, err
	}

	respTable := resp.(*glua.LTable)
	bodyVal := respTable.RawGetString("body")
	bodyStr := bodyVal.String()
	var parsed interface{}
	if err := json.Unmarshal([]byte(bodyStr), &parsed); err != nil {
		return glua.LNil, fmt.Errorf("invalid JSON response: %w", err)
	}
	return parseGoValueToLua(L, parsed)
}

func (c *httpClient) doRequest(L *glua.LState, opts *glua.LTable) (glua.LValue, error) {
	return c.doRequestWithBody(L, opts, nil)
}

func (c *httpClient) doRequestWithBody(L *glua.LState, opts *glua.LTable, body io.Reader) (glua.LValue, error) {
	method := strings.ToUpper(getStringField(opts, "method"))
	if method == "" {
		method = "GET"
	}
	rawURL := getStringField(opts, "url")
	if rawURL == "" {
		rawURL = getStringField(opts, "path")
	}
	if rawURL == "" {
		return glua.LNil, fmt.Errorf("request: path or url is required")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return glua.LNil, err
	}
	if parsed.Scheme == "" {
		if c.baseURL == nil {
			return glua.LNil, fmt.Errorf("request: base_url is required when path is relative")
		}
		parsed, err = c.baseURL.Parse(rawURL)
		if err != nil {
			return glua.LNil, err
		}
	}
	if qtbl := opts.RawGetString("query"); qtbl != glua.LNil {
		queryVals := parseQuery(qtbl)
		values := parsed.Query()
		for k, list := range queryVals {
			for _, v := range list {
				values.Add(k, v)
			}
		}
		parsed.RawQuery = values.Encode()
	}

	req, err := http.NewRequest(method, parsed.String(), body)
	if err != nil {
		return glua.LNil, err
	}

	headers := mergeHeaders(c.headers, parseHeaders(opts, "headers"))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	timeout := c.timeout
	if val := opts.RawGetString("timeout_ms"); val != glua.LNil {
		timeout = time.Duration(getNumberFromValue(val)) * time.Millisecond
	}

	client := &http.Client{}
	if timeout > 0 {
		client.Timeout = timeout
	}

	resp, err := client.Do(req)
	if err != nil {
		return glua.LNil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return glua.LNil, err
	}

	bodyStr := string(respBody)
	statusText := statusTextFromResponse(resp)
	throwOnHTTPError := c.throwOnHTTPError
	if val := opts.RawGetString("throw_on_http_error"); val != glua.LNil {
		throwOnHTTPError = val == glua.LTrue
	}
	if throwOnHTTPError && resp.StatusCode >= 400 {
		return glua.LNil, &httpStatusError{
			Status:     resp.StatusCode,
			StatusText: statusText,
			Body:       bodyStr,
		}
	}

	respTable := L.NewTable()
	respTable.RawSetString("status", glua.LNumber(resp.StatusCode))
	respTable.RawSetString("status_text", glua.LString(statusText))
	respTable.RawSetString("headers", headersToLuaTable(L, resp.Header))
	respTable.RawSetString("body", glua.LString(bodyStr))
	respTable.RawSetString("json", L.NewFunction(func(L *glua.LState) int {
		var parsed interface{}
		if err := json.Unmarshal([]byte(bodyStr), &parsed); err != nil {
			return pushLuaError(L, fmt.Errorf("invalid JSON response: %w", err))
		}
		luaVal, err := parseGoValueToLua(L, parsed)
		if err != nil {
			return pushLuaError(L, err)
		}
		L.Push(luaVal)
		L.Push(glua.LNil)
		return 2
	}))

	return respTable, nil
}

func checkHTTPClient(L *glua.LState) *httpClient {
	ud := L.CheckUserData(1)
	client, ok := ud.Value.(*httpClient)
	if !ok || client == nil {
		L.ArgError(1, "expected http client")
		return nil
	}
	return client
}

func parseHeaders(tbl *glua.LTable, key string) map[string]string {
	if tbl == nil {
		return nil
	}
	val := tbl.RawGetString(key)
	if val == glua.LNil {
		return nil
	}
	lt, ok := val.(*glua.LTable)
	if !ok {
		return nil
	}
	out := make(map[string]string)
	lt.ForEach(func(k, v glua.LValue) {
		out[k.String()] = v.String()
	})
	return out
}

func headersToLuaTable(L *glua.LState, headers http.Header) *glua.LTable {
	tbl := L.NewTable()
	for k, vals := range headers {
		arr := L.CreateTable(len(vals), 0)
		for i, v := range vals {
			arr.RawSetInt(i+1, glua.LString(v))
		}
		tbl.RawSetString(strings.ToLower(k), arr)
	}
	return tbl
}

func headersMapToLuaTable(L *glua.LState, headers map[string]string) *glua.LTable {
	tbl := L.NewTable()
	for k, v := range headers {
		tbl.RawSetString(k, glua.LString(v))
	}
	return tbl
}

func mergeHeaders(base, extra map[string]string) map[string]string {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func parseQuery(val glua.LValue) map[string][]string {
	out := make(map[string][]string)
	tbl, ok := val.(*glua.LTable)
	if !ok {
		return out
	}
	tbl.ForEach(func(k, v glua.LValue) {
		key := k.String()
		switch v := v.(type) {
		case *glua.LTable:
			var values []string
			v.ForEach(func(_, item glua.LValue) {
				values = append(values, item.String())
			})
			out[key] = values
		default:
			out[key] = []string{v.String()}
		}
	})
	return out
}

func getStringField(tbl *glua.LTable, key string) string {
	if tbl == nil {
		return ""
	}
	val := tbl.RawGetString(key)
	if val == glua.LNil {
		return ""
	}
	return val.String()
}

func getNumberField(tbl *glua.LTable, key string) float64 {
	if tbl == nil {
		return 0
	}
	val := tbl.RawGetString(key)
	return getNumberFromValue(val)
}

func getNumberFromValue(val glua.LValue) float64 {
	if val == glua.LNil {
		return 0
	}
	if n, ok := val.(glua.LNumber); ok {
		return float64(n)
	}
	return 0
}

func getBoolField(tbl *glua.LTable, key string) bool {
	if tbl == nil {
		return false
	}
	val := tbl.RawGetString(key)
	return val == glua.LTrue
}

func pushLuaError(L *glua.LState, err error) int {
	errTable := L.NewTable()
	errTable.RawSetString("message", glua.LString(err.Error()))
	if httpErr, ok := err.(*httpStatusError); ok {
		errTable.RawSetString("status", glua.LNumber(httpErr.Status))
		errTable.RawSetString("status_text", glua.LString(httpErr.StatusText))
		errTable.RawSetString("body", glua.LString(httpErr.Body))
	}
	L.Push(glua.LNil)
	L.Push(errTable)
	return 2
}

type httpStatusError struct {
	Status     int
	StatusText string
	Body       string
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("HTTP %d %s", e.Status, e.StatusText)
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
