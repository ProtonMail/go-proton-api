package server

import (
	"net/http"

	"github.com/ProtonMail/go-proton-api"
	"github.com/gin-gonic/gin"
)

func (s *Server) handleGetContactsEmails() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ContactEmails": []proton.ContactEmail{},
		})
	}
}
