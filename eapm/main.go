package main

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"time"

	"go.elastic.co/apm/module/apmhttp/v2"
	"go.elastic.co/apm/module/apmzap/v2"
	"go.uber.org/zap"
)

const url = "http://example.com"

func main() {
	logger, _ := zap.NewProduction(zap.WrapCore((&apmzap.Core{}).WrapCore))
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Infow("failed to fetch URL",
		// Structured context as loosely typed key-value pairs.
		"url", url,
		"attempt", 3,
		"backoff", time.Second,
	)

	defer func() {
		r := recover()
		sugar.Fatalw("panic detected", "error", r)
	}()

	// Instrument the default HTTP transport, so that outgoing
	// (reverse-proxy) requests are reported as spans.
	http.DefaultTransport = apmhttp.WrapRoundTripper(http.DefaultTransport, apmhttp.WithClientTrace())

	http.HandleFunc("/foo", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			r := recover()
			sugar.Errorw("panic detected", "error", r)
		}()
		panic("foo")
	})

	http.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	//#sugar.Fatal(http.ListenAndServe(":8080", nil))
	sugar.Info("starting server on localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if errors.Is(err, http.ErrServerClosed) {
		sugar.Infoln("server closed")
	} else if err != nil {
		sugar.Fatalw("error starting server", "error", err)
	}
}
