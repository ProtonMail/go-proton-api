package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/henrybear327/go-proton-api"
)

func (s *Server) handlePostDataStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req proton.SendStatsReq

		if err := c.BindJSON(&req); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		if !validateSendStatReq(&req) {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"Code": proton.SuccessCode,
		})
	}
}

func (s *Server) handlePostDataStatsMultiple() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req proton.SendStatsMultiReq

		if err := c.BindJSON(&req); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		for _, event := range req.EventInfo {
			if !validateSendStatReq(&event) {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"Code": proton.SuccessCode,
		})
	}
}

func validateSendStatReq(req *proton.SendStatsReq) bool {
	return req.MeasurementGroup != ""
}

func (s *Server) handleObservabilityPost() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req proton.ObservabilityBatch
		if err := c.BindJSON(&req); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		s.b.PushObservabilityMetrics(req.Metrics)

		c.JSON(http.StatusOK, gin.H{
			"Code": proton.SuccessCode,
		})
	}
}
