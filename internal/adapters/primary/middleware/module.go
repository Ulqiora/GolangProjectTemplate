package middleware

import (
	"context"
	"net/http"
	"strings"

	"GolangTemplateProject/pkg/jwt"
	"github.com/gin-gonic/gin"
)

const (
	UserContextKey = "user"
)

type Middleware interface {
	ValidateCredential()
}

type Module struct {
	jwtManager jwt.JWTManager
}

func (m *Module) ValidateCredential() gin.HandlerFunc {
	return func(c *gin.Context) {
		creadential := c.GetString(UserContextKey)
		if creadential == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		parts := strings.Split(creadential, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Header("WWW-Authenticate", "invalid authorization header")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims, err := m.jwtManager.Verify(parts[1])
		if err != nil {
			c.Header("WWW-Authenticate", "invalid authorization header")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(c.Request.Context(), UserContextKey, claims)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
