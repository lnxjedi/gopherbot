package javascript

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// HttpRequestOptions represents options for the HTTP request.
type HttpRequestOptions struct {
	Headers map[string]string `js:"headers"`
	Timeout int               `js:"timeout"` // Timeout in seconds
}

// HttpResponse represents the HTTP response.
type HttpResponse struct {
	Status     string              `js:"status"`
	StatusCode int                 `js:"statusCode"`
	Headers    map[string][]string `js:"headers"`
	Body       string              `js:"body"`
}

// HttpClient represents the HTTP client.
type HttpClient struct {
	BaseURI string
	Options HttpRequestOptions
	vm      *goja.Runtime
}

// Get performs an HTTP GET request.
func (c *HttpClient) Get(path string, options *HttpRequestOptions, callback func(response *HttpResponse, err interface{})) {
	c.makeRequest("GET", path, "", options, callback)
}

// Post performs an HTTP POST request.
func (c *HttpClient) Post(path string, body string, options *HttpRequestOptions, callback func(response *HttpResponse, err interface{})) {
	c.makeRequest("POST", path, body, options, callback)
}

// Put performs an HTTP PUT request.
func (c *HttpClient) Put(path string, body string, options *HttpRequestOptions, callback func(response *HttpResponse, err interface{})) {
	c.makeRequest("PUT", path, body, options, callback)
}

// Delete performs an HTTP DELETE request.
func (c *HttpClient) Delete(path string, options *HttpRequestOptions, callback func(response *HttpResponse, err interface{})) {
	c.makeRequest("DELETE", path, "", options, callback)
}

// makeRequest is a helper function to perform the HTTP request.
func (c *HttpClient) makeRequest(method string, path string, body string, options *HttpRequestOptions, callback func(response *HttpResponse, err interface{})) {
	// Default to client options if request options are not provided
	if options == nil {
		options = &c.Options
	}

	// Construct the full URL
	fullURL, err := url.Parse(c.BaseURI)
	if err != nil {
		callback(nil, c.vm.ToValue(err))
		return
	}
	fullURL, err = fullURL.Parse(path)
	if err != nil {
		callback(nil, c.vm.ToValue(err))
		return
	}

	// Create the HTTP request
	req, err := http.NewRequest(method, fullURL.String(), strings.NewReader(body))
	if err != nil {
		callback(nil, c.vm.ToValue(err))
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
		callback(nil, c.vm.ToValue(err))
		return
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		callback(nil, c.vm.ToValue(err))
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
	callback(response, nil)
}

// Http represents the http module for JavaScript.
type Http struct {
	vm *goja.Runtime
}

// New creates a new HttpClient instance.
func (h *Http) New(baseURI string, options *HttpRequestOptions) *HttpClient {
	return &HttpClient{
		BaseURI: baseURI,
		Options: *options,
		vm:      h.vm,
	}
}

// addHttpHandler adds the http module to the Goja VM.
func addHttpHandler(vm *goja.Runtime) {
	httpModule := &Http{vm: vm}
	vm.Set("http", map[string]interface{}{
		"New": httpModule.New,
	})
}
