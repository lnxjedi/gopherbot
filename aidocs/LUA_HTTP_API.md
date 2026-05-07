# Lua HTTP API

Lua exposes the upstream GopherLua HTTP module from
`github.com/cjoudrey/gluahttp` as `require("http")`.

Gopherbot also exposes a small native `json` module with `encode(value)` and
`decode(string)` helpers because gluahttp intentionally returns response bodies
as strings.

## Basic GET

```lua
local http = require("http")
local json = require("json")

local response, err = http.get("https://api.example.com/v1/status", {
  query = "region=us-east-1",
  headers = {
    Accept = "application/json",
  },
  timeout = "10s",
})
if err then
  return err
end

if response.status_code >= 400 then
  return "HTTP " .. tostring(response.status_code) .. ": " .. response.body
end

local data, decodeErr = json.decode(response.body)
if decodeErr then
  return decodeErr
end
```

## JSON POST

```lua
local http = require("http")
local json = require("json")

local body, encodeErr = json.encode({ name = "foo" })
if encodeErr then
  return encodeErr
end

local response, err = http.post("https://api.example.com/v1/items", {
  body = body,
  headers = {
    Accept = "application/json",
    ["Content-Type"] = "application/json",
  },
  timeout = "10s",
})
```

## HTTP Helpers

gluahttp provides:

- `http.get(url, options)`
- `http.delete(url, options)`
- `http.head(url, options)`
- `http.patch(url, options)`
- `http.post(url, options)`
- `http.put(url, options)`
- `http.request(method, url, options)`
- `http.request_batch(requests)`

## Request Options

- `query` (string) - URL-encoded query string.
- `cookies` (table) - Cookie names and values.
- `headers` (table) - Request headers.
- `body` (string) - Request body.
- `form` (string) - Deprecated URL-encoded request body; sets
  `Content-Type: application/x-www-form-urlencoded`.
- `timeout` (number or string) - Number of seconds or Go-style duration string
  such as `"500ms"` or `"1m"`.
- `auth` (table) - Basic auth table: `{ user = "name", pass = "secret" }`.

## Response

Successful HTTP transport returns an `http.response` userdata:

```lua
{
  body = "...",
  body_size = 123,
  headers = { ["Content-Type"] = "application/json" },
  cookies = { session = "..." },
  status_code = 200,
  url = "https://api.example.com/v1/status"
}
```

HTTP 4xx/5xx responses are not Lua errors. Check `response.status_code`
explicitly when the caller wants to treat non-2xx/3xx statuses as failures.

Transport failures, invalid URLs, invalid timeout values, and request context
timeouts return `(nil, error_message)`.

## JSON Helpers

```lua
local json = require("json")

local encoded, err = json.encode({ enabled = true })
local decoded, err = json.decode('{"enabled":true}')
```

Both helpers return `(nil, error_message)` on failure.
