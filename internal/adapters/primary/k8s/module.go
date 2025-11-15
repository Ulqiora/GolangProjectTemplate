package k8s

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ModulePrometheus struct {
	router gin.IRouter
}

func (m *ModulePrometheus) Register(app gin.IRouter) {
	m.router = app
	app.GET("/", m.Liveness)
}

func NewModulePrometheus() *ModulePrometheus {
	return &ModulePrometheus{}
}

func (m *ModulePrometheus) Liveness(ctx *gin.Context) {
	ctx.Writer.WriteHeader(http.StatusNoContent)
}

func (m *ModulePrometheus) Readiness(ctx *gin.Context) {
	ctx.Writer.WriteHeader(http.StatusNoContent)
}
