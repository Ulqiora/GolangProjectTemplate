package prometheus

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type FiberAdapter struct {
	ctx           *fiber.Ctx
	header        http.Header
	headerSent    bool
	statusCode    int
	statusCodeSet bool
}

func (f *FiberAdapter) Header() http.Header {
	if f.header == nil {
		f.header = make(http.Header)
	}

	return f.header
}

func (f *FiberAdapter) Write(b []byte) (int, error) {
	f.writeHeaders()
	return f.ctx.Write(b)
}

func (f *FiberAdapter) WriteHeader(statusCode int) {
	f.statusCode = statusCode
	f.statusCodeSet = true
	f.writeHeaders()
	f.ctx.Status(statusCode)
}

func (f *FiberAdapter) writeHeaders() {
	if f.headerSent {
		return
	}
	f.headerSent = true
	for key, values := range f.Header() {
		for _, value := range values {
			f.ctx.Append(key, value)
		}
	}
	if f.statusCodeSet {
		f.ctx.Status(f.statusCode)
	}
}

func Handler() http.Handler {
	return promhttp.Handler()
}

func HandlerFor(gatherer prometheus.Gatherer, opts promhttp.HandlerOpts) http.Handler {
	return promhttp.HandlerFor(gatherer, opts)
}

func FiberHandler() fiber.Handler {
	return FiberHandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})
}

func FiberHandlerFor(gatherer prometheus.Gatherer, opts promhttp.HandlerOpts) fiber.Handler {
	handler := HandlerFor(gatherer, opts)

	return func(ctx *fiber.Ctx) error {
		request := &http.Request{
			Method: string(ctx.Request().Header.Method()),
			Header: make(http.Header),
			Body:   http.NoBody,
		}

		handler.ServeHTTP(&FiberAdapter{ctx: ctx}, request)
		return nil
	}
}
