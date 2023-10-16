package server

import (
	"net/http"

	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/go-proton-api/server/backend"
	"github.com/gin-gonic/gin"
)

func (s *Server) handleGetContacts() gin.HandlerFunc {
	return func(c *gin.Context) {
		contacts, err := s.b.GetUserContacts(c.GetString("UserID"))
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"Code":          proton.MultiCode,
			"ContactEmails": contacts,
		})
	}
}

func (s *Server) handleGetContactsEmails() gin.HandlerFunc {
	return func(c *gin.Context) {
		contacts, err := s.b.GetUserContactEmails(c.GetString("UserID"), c.GetString("email"))
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"Code":          proton.MultiCode,
			"ContactEmails": contacts,
		})
	}
}

func (s *Server) handlePostContacts() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req proton.CreateContactsReq
		err := c.BindJSON(&req)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		var responses []proton.CreateContactsRes

		userId := c.GetString("UserID")
		user, err := s.b.GetUser(userId)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		pubKeys, err := s.b.GetPublicKeys(user.Email)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		userKr, err := proton.PublicKeys(pubKeys).GetKeyRing()
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		for i, v := range req.Contacts {
			for _, card := range v.Cards {
				contactID, err := s.b.GenerateContactID(userId)
				if err != nil {
					responses = append(responses,
						proton.CreateContactsRes{
							Index: i,
							Response: proton.CreateContactResp{
								Response: proton.APIError{Code: proton.InvalidValue, Message: err.Error()},
							},
						})
					continue
				}

				contact, err := backend.ContactCardToContact(card, contactID, userKr)
				if err != nil {
					responses = append(responses,
						proton.CreateContactsRes{
							Index: i,
							Response: proton.CreateContactResp{
								Response: proton.APIError{Code: proton.InvalidValue, Message: err.Error()},
							},
						})
					continue
				}
				contact, err = s.b.AddUserContact(userId, contact)
				if err != nil {
					responses = append(responses,
						proton.CreateContactsRes{
							Index: i,
							Response: proton.CreateContactResp{
								Response: proton.APIError{Code: proton.InvalidValue, Message: err.Error()},
							},
						})
					continue
				}
				responses = append(responses,
					proton.CreateContactsRes{
						Index: i,
						Response: proton.CreateContactResp{
							Response: proton.APIError{Code: proton.SuccessCode},
							Contact:  contact,
						},
					})
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"Code":      proton.MultiCode,
			"Responses": responses,
		})
	}
}

func (s *Server) handleGetContact() gin.HandlerFunc {
	return func(c *gin.Context) {
		contact, err := s.b.GetUserContact(c.GetString("UserID"), c.Param("contactID"))
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"Code":    proton.MultiCode,
			"Contact": contact,
		})
	}
}

func (s *Server) handlePutContact() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req proton.UpdateContactReq
		err := c.BindJSON(&req)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		contact, err := s.b.UpdateUserContact(c.GetString("UserID"), c.Param("contactID"), req.Cards)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"Code":    proton.MultiCode,
			"Contact": contact,
		})
	}
}
