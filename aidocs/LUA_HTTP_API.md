# Lua HTTP Compatibility

`require("http")` exposes the synchronous `gluahttp`-style surface, and
`require("json")` supplies encode/decode helpers. Current options and response
fields are defined by the Lua module source and its full integration suite.

HTTP 4xx/5xx responses are returned normally; callers inspect `status_code`.
Transport, URL, and timeout failures return `(nil, error_message)`.
