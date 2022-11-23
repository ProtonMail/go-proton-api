package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleGetMailSettings() gin.HandlerFunc {
	return func(c *gin.Context) {
		settings, err := s.b.GetMailSettings(c.GetString("UserID"))
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"MailSettings": settings,
		})
	}
}
