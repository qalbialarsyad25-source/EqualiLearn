package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
)

func (m *Middleware) Authentication(c *gin.Context) {
	header := c.GetHeader("Authorization")

	if header == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		c.Abort()
		return
	}

	parts := strings.Split(header, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "invalid authorization format",
		})
		return
	}

	token := parts[1]
	userIdStr, role, err := m.jwt.ValidateToken(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "invalid token/failed to validate token",
		})
		return
	}

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid",
		})
		return
	}

	c.Set("userId", userId)
	c.Set("role", role)
	c.Next()
}
