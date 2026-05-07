# JavaScript HTTP API

JavaScript exposes a native synchronous HTTP module as `require("http")`.

The API intentionally mirrors Lua's `require("http")` surface while using
JavaScript-friendly response field names and request body handling.

## Basic GET

```js
const http = require("http");

const response = http.get("https://api.example.com/v1/status", {
  query: { region: "us-east-1" },
  headers: {
    Accept: "application/json",
  },
  timeout: "10s",
});

if (!response.ok) {
  throw new Error(`HTTP ${response.statusCode}: ${response.body}`);
}

const data = response.json;
```

## JSON POST

```js
const http = require("http");

const response = http.post("https://api.example.com/v1/items", {
  body: { name: "foo" },
  headers: {
    Accept: "application/json",
  },
  timeout: "10s",
});
```

Object and array request bodies are encoded as JSON automatically. If the caller
does not set `Content-Type`, the runtime sets `Content-Type: application/json`
for JSON-encoded bodies.

## HTTP Helpers

- `http.get(url, options)`
- `http.delete(url, options)`
- `http.head(url, options)`
- `http.patch(url, options)`
- `http.post(url, options)`
- `http.put(url, options)`
- `http.request(method, url, options)`
- `http.requestBatch(requests)` / `http.request_batch(requests)`

## Request Options

- `query` (object or string) - Query parameters. Objects are URL-encoded; string
  values are used as the raw query string.
- `cookies` (object) - Cookie names and values.
- `headers` (object) - Request headers.
- `body` (string, byte array, object, or array) - Request body. Objects and
  arrays are JSON-encoded automatically.
- `form` (string) - URL-encoded request body; sets
  `Content-Type: application/x-www-form-urlencoded` when `body` is not set.
- `timeout` (number or string) - Number of seconds or Go-style duration string
  such as `"500ms"` or `"1m"`.
- `timeoutMs` (number) - Millisecond timeout convenience alias.
- `auth` (object) - Basic auth object: `{ user: "name", pass: "secret" }`.

## Response

Successful HTTP transport returns:

```js
{
  body: "...",
  bodySize: 123,
  headers: { "content-type": "application/json" },
  cookies: { session: "..." },
  statusCode: 200,
  statusText: "OK",
  ok: true,
  url: "https://api.example.com/v1/status",
  json: { ... }
}
```

`json` is automatically parsed when the response `Content-Type` is
`application/json` or ends in `+json`. If parsing fails, or the content type is
not JSON, `json` is `null` and `body` remains the original response string.

HTTP 4xx/5xx responses are not JavaScript exceptions. Check `response.ok` or
`response.statusCode` explicitly when the caller wants to treat non-2xx/3xx
statuses as failures.

Transport failures, invalid URLs, invalid timeout values, and request context
timeouts throw JavaScript `Error` values.

## Runtime Note

The embedded JavaScript runtime is synchronous and does not support Node's
streaming asynchronous `http` module. Gopherbot's `http` module blocks until the
request completes and returns the full response body.
