# Lua HTTP API

This describes the Lua HTTP helper module exposed as `require("gopherbot_http")`.

## API shape (Lua style)

```lua
local http = require("gopherbot_http")
local client, err = http.create_client{
  base_url = "https://api.example.com",
  headers = { Authorization = "Bearer " .. token },
  timeout_ms = 10000,
  throw_on_http_error = false,
}

local resp, err = client:request{
  method = "GET",
  path = "/v1/status",
  query = { region = "us-east-1" },
}

local data, err = client:get_json("/v1/items")
local created, err = client:post_json("/v1/items", { name = "foo" })
local updated, err = client:put_json("/v1/items/1", { name = "bar" })
```

## Client options

- `base_url` (string, optional) - Base URL for relative paths.
- `headers` (table) - Default headers for all requests.
- `timeout_ms` (number) - Default timeout in milliseconds.
- `throw_on_http_error` (boolean) - Return error on non-2xx/3xx responses when true.

## Request options

- `method` (string) - HTTP method; defaults to `GET`.
- `path` (string) - Relative path (requires `base_url`) or absolute URL.
- `url` (string) - Absolute URL (overrides `path`).
- `query` (table) - Query string parameters; values are stringified.
- `headers` (table) - Per-request headers.
- `body` (string) - Request body for non-JSON requests.
- `timeout_ms` (number) - Per-request timeout in milliseconds.
- `throw_on_http_error` (boolean) - Override default error handling.

## Response table

```lua
{
  status = 200,
  status_text = "OK",
  headers = { ["content-type"] = { "application/json" } },
  body = "...",
  json = function() return <lua table>, err end
}
```

## Errors

All calls return `(result, err)` where `err` is a table:

```lua
{
  message = "HTTP 500 Internal Server Error",
  status = 500,
  status_text = "Internal Server Error",
  body = "..."
}
```
