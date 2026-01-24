# JavaScript HTTP API (Design Notes)

This doc captures the current JavaScript HTTP support and the goals for a redesigned, devops-friendly API. Use it to restart context and drive implementation.

## Current implementation (baseline)

The legacy API (`http.New(...)` with callbacks) has been replaced. See **Current JS API** below for the new surface.

## Design goals for the new API

Prioritize a first-class JS developer experience and DevOps usage:

- **Ergonomic, modern JS API** with consistent error handling.
- **Simple JSON helpers** (`getJSON`, `postJSON`, and a `json()` response helper).
- **Clean error semantics**: network errors throw; HTTP error handling opt-in (e.g., `throwOnHTTPError`).
- **Timeouts and cancellation**: per-request timeout in ms; future support for cancel tokens.
- **Defaults**: base URL, default headers, and default timeout set on a client object.
- **Minimal surface**: one `request` function and a few helpers; no heavy wrappers.
- **No backward-compat constraint**: v3 is not released; we can replace the existing API.

## Current JS API (implemented)

```js
const http = require("gopherbot_http");
const client = http.createClient({
  baseURL: "https://api.example.com",
  headers: { Authorization: `Bearer ${token}` },
  timeoutMs: 10000,
  throwOnHTTPError: false,
});

const res = client.request({
  method: "GET",
  path: "/v1/status",
  query: { region: "us-east-1" },
});

const data = client.getJSON("/v1/items");
const created = client.postJSON("/v1/items", { name: "foo" });
```

**Client options:**
- `baseURL` (string, optional) - Base URL for relative paths.
- `headers` (object) - Default headers for all requests.
- `timeoutMs` (number) - Default timeout in milliseconds.
- `throwOnHTTPError` (boolean) - Throw on non-2xx/3xx responses when true.

**Request options:**
- `method` (string) - HTTP method; defaults to GET if empty.
- `path` (string) - Relative path (requires `baseURL`) or absolute URL.
- `url` (string) - Absolute URL (overrides `path`).
- `query` (object) - Query string parameters; values are stringified.
- `headers` (object) - Per-request headers.
- `body` (string or byte array) - Request body for non-JSON requests.
- `timeoutMs` (number) - Per-request timeout in milliseconds.
- `throwOnHTTPError` (boolean) - Override default error handling.

**Response shape:**
```js
{
  status: 200,
  statusText: "OK",
  headers: { "content-type": ["application/json"] },
  body: "...",
  json: () => ({ ... }) // throws on invalid JSON
}
```

**JSON helpers:**
- `client.getJSON(path, options)` returns parsed JSON.
- `client.postJSON(path, payload, options)` sends JSON and returns parsed JSON.

**Errors:**
- Network errors throw `Error` with message + optional cause.
- If `throwOnHTTPError` is true, non-2xx responses throw an error with `status`/`body`.

**Availability:**
- The `gopherbot_http` module is registered alongside `require()` setup, so it is available in both runtime and `configure` scripts.

**Runtime note:**
- The embedded JS runtime does not support Promises/`async` today, so HTTP helpers are synchronous.

## Open questions

- Should we also expose `gopherbot_http` via a global (e.g., `GBOT.http`) for convenience?
- Should we support streaming bodies or only string/JSON?
- Do we need retry/backoff helpers, or keep those in user space?

## Next steps

- Decide on final JS API surface.
- Implement in `modules/javascript/http_object.go` and expose into the VM.
- Update tests in `test/jsfull` to exercise the new API.
- Update `aidocs/JS_METHOD_CHECKLIST.md` once the HTTP API is finalized.
