package prometheus

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ModulePrometheus struct {
	router gin.IRouter
}

func (m *ModulePrometheus) Register(app gin.IRouter) {
	m.router = app
	app.GET("/", gin.WrapH(promhttp.Handler()))
}

func NewModulePromehteus() *ModulePrometheus {
	return &ModulePrometheus{}
}
