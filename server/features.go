package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleGetFeatures() gin.HandlerFunc {
	return func(c *gin.Context) {
		ff := s.b.GetFeatureFlags()
		c.JSON(http.StatusOK, gin.H{
			"Code":    1000,
			"Toggles": ff,
		})
	}
}
