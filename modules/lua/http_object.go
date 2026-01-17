package lua

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yuin/gopher-lua"
)

// HttpRequestOptions represents options for the HTTP request.
type HttpRequestOptions struct {
	Headers map[string]string
	Timeout int // Timeout in seconds
}

// HttpResponse represents the HTTP response.
type HttpResponse struct {
	Status     string
	StatusCode int
	Headers    map[string][]string
	Body       string
}

// HttpClient represents the HTTP client.
type HttpClient struct {
	BaseURI string
	Options HttpRequestOptions
	ls      *lua.LState
}

// Get performs an HTTP GET request.
func (c *HttpClient) Get(L *lua.LState) int {
	path := L.CheckString(2)
	var body string
	optionsTable := L.OptTable(3, nil)
	callback := L.CheckFunction(4)

	options := c.Options
	if optionsTable != nil {
		options.extractFromTable(optionsTable)
	}

	c.makeRequest("GET", path, body, &options, callback)
	return 0
}

// Post performs an HTTP POST request.
func (c *HttpClient) Post(L *lua.LState) int {
	path := L.CheckString(2)
	body := L.CheckString(3)
	optionsTable := L.OptTable(4, nil)
	callback := L.CheckFunction(5)

	options := c.Options
	if optionsTable != nil {
		options.extractFromTable(optionsTable)
	}

	c.makeRequest("POST", path, body, &options, callback)
	return 0
}

// Put performs an HTTP PUT request.
func (c *HttpClient) Put(L *lua.LState) int {
	path := L.CheckString(2)
	body := L.CheckString(3)
	optionsTable := L.OptTable(4, nil)
	callback := L.CheckFunction(5)

	options := c.Options
	if optionsTable != nil {
		options.extractFromTable(optionsTable)
	}

	c.makeRequest("PUT", path, body, &options, callback)
	return 0
}

// Delete performs an HTTP DELETE request.
func (c *HttpClient) Delete(L *lua.LState) int {
	path := L.CheckString(2)
	optionsTable := L.OptTable(3, nil)
	callback := L.CheckFunction(4)

	options := c.Options
	if optionsTable != nil {
		options.extractFromTable(optionsTable)
	}

	c.makeRequest("DELETE", path, "", &options, callback)
	return 0
}

// extractFromTable extracts HttpRequestOptions from a Lua table.
func (o *HttpRequestOptions) extractFromTable(tbl *lua.LTable) {
	// Extract Headers
	headersTable, ok := tbl.RawGetString("headers").(*lua.LTable)
	if ok && headersTable != nil {
		o.Headers = make(map[string]string)
		headersTable.ForEach(func(key, value lua.LValue) {
			o.Headers[key.String()] = value.String()
		})
	}

	// Extract Timeout
	timeout := tbl.RawGetString("timeout")
	if timeout.Type() == lua.LTNumber {
		o.Timeout = int(timeout.(lua.LNumber))
	}
}

// makeRequest is a helper function to perform the HTTP request.
func (c *HttpClient) makeRequest(method string, path string, body string, options *HttpRequestOptions, callback *lua.LFunction) {
	// Construct the full URL
	fullURL, err := url.Parse(c.BaseURI)
	if err != nil {
		c.callErrorCallback(callback, err)
		return
	}
	fullURL, err = fullURL.Parse(path)
	if err != nil {
		c.callErrorCallback(callback, err)
		return
	}

	// Create the HTTP request
	req, err := http.NewRequest(method, fullURL.String(), strings.NewReader(body))
	if err != nil {
		c.callErrorCallback(callback, err)
		return
	}

	// Add headers to the request
	for key, value := range c.Options.Headers {
		req.Header.Add(key, value)
	}
	if options != nil {
		for key, value := range options.Headers {
			req.Header.Add(key, value)
		}
	}

	// Create a new HTTP client with timeout if specified
	client := &http.Client{}
	if options.Timeout > 0 {
		client.Timeout = time.Duration(options.Timeout) * time.Second
	}

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		c.callErrorCallback(callback, err)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.callErrorCallback(callback, err)
		return
	}

	// Create the response object
	response := &HttpResponse{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       string(respBody),
	}

	// Call the callback with the response
	c.callResponseCallback(callback, response)
}

func (c *HttpClient) callErrorCallback(callback *lua.LFunction, err error) {
	c.ls.CallByParam(lua.P{
		Fn:      callback,
		NRet:    0,
		Protect: true,
	}, lua.LNil, lua.LString(err.Error()))
}

func (c *HttpClient) callResponseCallback(callback *lua.LFunction, response *HttpResponse) {
	respTable := c.ls.NewTable()
	respTable.RawSetString("status", lua.LString(response.Status))
	respTable.RawSetString("statusCode", lua.LNumber(response.StatusCode))
	respTable.RawSetString("body", lua.LString(response.Body))

	headersTable := c.ls.NewTable()
	for k, v := range response.Headers {
		valTable := c.ls.NewTable()
		for _, s := range v {
			valTable.Append(lua.LString(s))
		}
		headersTable.RawSetString(k, valTable)
	}
	respTable.RawSetString("headers", headersTable)

	c.ls.CallByParam(lua.P{
		Fn:      callback,
		NRet:    0,
		Protect: true,
	}, respTable, lua.LNil)
}

// Constructor for HttpClient
func newHttpClient(L *lua.LState) int {
	baseURI := L.CheckString(1)
	optionsTable := L.OptTable(2, nil)

	options := HttpRequestOptions{}
	if optionsTable != nil {
		options.extractFromTable(optionsTable)
	}

	client := &HttpClient{
		BaseURI: baseURI,
		Options: options,
		ls:      L,
	}

	ud := L.NewUserData()
	ud.Value = client
	L.SetMetatable(ud, L.GetTypeMetatable("http_client"))
	L.Push(ud)
	return 1
}

// addHttpHandler adds the http module to the Lua state.
func addHttpHandler(L *lua.LState) {
	// Metatable for HttpClient objects
	mt := L.NewTypeMetatable("http_client")
	L.SetGlobal("http_client", mt)
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"get":    httpGet,
		"post":   httpPost,
		"put":    httpPut,
		"delete": httpDelete,
	}))

	// http module table
	httpModule := L.NewTable()
	L.SetFuncs(httpModule, map[string]lua.LGFunction{
		"new": newHttpClient,
	})
	L.SetGlobal("http", httpModule)
}

func httpGet(L *lua.LState) int {
	ud := L.CheckUserData(1)
	client, ok := ud.Value.(*HttpClient)
	if !ok {
		L.ArgError(1, "http_client expected")
	}
	return client.Get(L)
}

func httpPost(L *lua.LState) int {
	ud := L.CheckUserData(1)
	client, ok := ud.Value.(*HttpClient)
	if !ok {
		L.ArgError(1, "http_client expected")
	}
	return client.Post(L)
}

func httpPut(L *lua.LState) int {
	ud := L.CheckUserData(1)
	client, ok := ud.Value.(*HttpClient)
	if !ok {
		L.ArgError(1, "http_client expected")
	}
	return client.Put(L)
}

func httpDelete(L *lua.LState) int {
	ud := L.CheckUserData(1)
	client, ok := ud.Value.(*HttpClient)
	if !ok {
		L.ArgError(1, "http_client expected")
	}
	return client.Delete(L)
}
