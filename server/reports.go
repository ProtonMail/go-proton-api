package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (s *Server) handlePostReportBug() gin.HandlerFunc {
	return func(c *gin.Context) {
		form, err := c.MultipartForm()
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		asyncAttach, err := strconv.ParseBool(form.Value["AsyncAttachments"][0])
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		if asyncAttach {
			token := s.b.CreateCSTicket()
			c.JSON(http.StatusOK, gin.H{
				"Token": token,
			})
		}
	}
}

func (s *Server) handlePostReportBugAttachments() gin.HandlerFunc {
	return func(c *gin.Context) {
		form, err := c.MultipartForm()
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		if !s.b.GetCSTicket(form.Value["Token"][0]) {
			_ = c.AbortWithError(http.StatusBadRequest, errors.New("Token not found in CS Ticket List"))
			return
		}
	}
}
