# JavaScript HTTP Compatibility

`require("http")` is a synchronous, buffered Gopherbot module, not Node's
streaming API. Current methods/options and response fields are defined in the
JavaScript module source and its full integration suite.

Object/array bodies are JSON-encoded. JSON responses are parsed when their
content type is JSON. HTTP 4xx/5xx responses are returned normally; callers
must inspect `ok`/`statusCode`. Transport, URL, and timeout failures throw.
