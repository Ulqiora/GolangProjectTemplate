package prometheus

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type FiberAdapter struct {
	ctx *fiber.Ctx
}

func (f *FiberAdapter) Header() http.Header {
	return http.Header{}
}

func (f *FiberAdapter) Write(b []byte) (int, error) {
	return f.ctx.Write(b)
}

func (f *FiberAdapter) WriteHeader(statusCode int) {
	f.ctx.Status(statusCode)
}
