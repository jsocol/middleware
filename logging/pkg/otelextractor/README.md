# otelextractor

Returns a `ContextExtractor` for use with `jsocol.io/middleware/logging`
that adds Trace and Span data from OpenTelemetry tracing state in the
context to the log record.

```go
package main

import (
	"net/http"

	"jsocol.io/middleware/logging"
	"jsocol.io/middleware/logging/pkg/otelextractor"
)

func main() {
	mux := http.NewServeMux()

	extractor := otelextractor.New(nil)
	wrapped := logging.Wrap(mux, logging.WithContextExtractors(extractor))

	http.ListenAndServe(":8000", wrapped)
}
```
