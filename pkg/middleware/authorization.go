package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (m *Middleware) Authorization(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles := c.GetString("role")

		for _, role := range roles {
			if userRoles == role {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized Role",
		})
	}
}
