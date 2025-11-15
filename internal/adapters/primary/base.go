package primary

import "github.com/gin-gonic/gin"

type Registrar interface {
	Register(app gin.IRouter)
}
