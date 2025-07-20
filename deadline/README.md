# Deadline Middleware

Package `deadline` contains middleware that provides deadline
propagation via HTTP headers.

## Quick Start

```go
import (
	"log"
	"net/http"
	"time"

	"jsocol.io/middleware/deadline"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("GET /long-running", &SomeLongRunningHandler{})

	wrapped := deadline.Wrap(mux, deadline.WithMaxTimeout(5*time.Second))

	if err := http.ListenAndServe("0:8080", wrapped); err != nil {
		log.Fatal(err)
	}
}
```
