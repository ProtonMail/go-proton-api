package server

import (
	"net/http"

	"github.com/ProtonMail/go-proton-api"
	"github.com/bradenaw/juniper/xslices"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

func (s *Server) handleGetAddresses() gin.HandlerFunc {
	return func(c *gin.Context) {
		addresses, err := s.b.GetAddresses(c.GetString("UserID"))
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"Addresses": addresses,
		})
	}
}

func (s *Server) handleGetAddress() gin.HandlerFunc {
	return func(c *gin.Context) {
		addresses, err := s.b.GetAddresses(c.GetString("UserID"))
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"Address": addresses[xslices.IndexFunc(addresses, func(address proton.Address) bool {
				return address.ID == c.Param("addressID")
			})],
		})
	}
}

func (s *Server) handlePutAddressesOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req proton.OrderAddressesReq

		if err := c.BindJSON(&req); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		addresses, err := s.b.GetAddresses(c.GetString("UserID"))
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		if len(req.AddressIDs) != len(addresses) {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		for _, address := range addresses {
			if !slices.Contains(req.AddressIDs, address.ID) {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}
		}

		if err := s.b.SetAddressOrder(c.GetString("UserID"), req.AddressIDs); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}
}
