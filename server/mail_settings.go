package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/henrybear327/go-proton-api"
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

func (s *Server) handlePutMailSettingsAttachPublicKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req proton.SetAttachPublicKeyReq

		if err := c.ShouldBindJSON(&req); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		settings, err := s.b.SetMailSettingsAttachPublicKey(c.GetString("UserID"), bool(req.AttachPublicKey))
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"MailSettings": settings,
		})
	}
}

func (s *Server) handlePutMailSettingsDraftType() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req proton.SetDraftMIMETypeReq

		if err := c.ShouldBindJSON(&req); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		settings, err := s.b.SetMailSettingsDraftMIMEType(c.GetString("UserID"), req.MIMEType)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"MailSettings": settings,
		})
	}
}

func (s *Server) handlePutMailSettingsSign() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req proton.SetSignExternalMessagesReq

		if err := c.ShouldBindJSON(&req); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		settings, err := s.b.SetMailSettingsSign(c.GetString("UserID"), req.Sign)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"MailSettings": settings,
		})
	}
}

func (s *Server) handlePutMailSettingsPGPScheme() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req proton.SetDefaultPGPSchemeReq

		if err := c.ShouldBindJSON(&req); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		settings, err := s.b.SetMailSettingsPGPScheme(c.GetString("UserID"), req.PGPScheme)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"MailSettings": settings,
		})
	}
}
