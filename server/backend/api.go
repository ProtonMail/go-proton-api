package backend

import (
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ProtonMail/gluon/rfc822"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/bradenaw/juniper/xslices"
	"github.com/henrybear327/go-proton-api"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func (b *Backend) GetUser(userID string) (proton.User, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) (proton.User, error) {
		return withAcc(b, userID, func(acc *account) (proton.User, error) {
			return withMessages(b, func(m map[string]*message) (proton.User, error) {
				return withAtts(b, func(attachments map[string]*attachment) (proton.User, error) {
					user := acc.toUser()

					var messageBytes uint64
					for _, v := range m {
						if _, ok := acc.addresses[v.addrID]; ok {
							messageBytes += uint64(len(v.armBody))
						}

						for _, a := range v.attIDs {
							if attach, ok := attachments[a]; ok {
								messageBytes += uint64(len(b.attData[attach.attDataID]))
							}
						}
					}

					user.ProductUsedSpace.Mail = messageBytes

					return user, nil
				})
			})
		})
	})
}

func (b *Backend) GetKeySalts(userID string) ([]proton.Salt, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) ([]proton.Salt, error) {
		return withAcc(b, userID, func(acc *account) ([]proton.Salt, error) {
			return xslices.Map(acc.keys, func(key key) proton.Salt {
				return proton.Salt{
					ID:      key.keyID,
					KeySalt: base64.StdEncoding.EncodeToString(acc.salt),
				}
			}), nil
		})
	})
}

func (b *Backend) GetMailSettings(userID string) (proton.MailSettings, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) (proton.MailSettings, error) {
		return withAcc(b, userID, func(acc *account) (proton.MailSettings, error) {
			return acc.mailSettings.toMailSettings(), nil
		})
	})
}

func (b *Backend) SetMailSettingsAttachPublicKey(userID string, attach bool) (proton.MailSettings, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.MailSettings, error) {
		return withAcc(b, userID, func(acc *account) (proton.MailSettings, error) {
			acc.mailSettings.attachPubKey = attach
			return acc.mailSettings.toMailSettings(), nil
		})
	})
}

func (b *Backend) SetMailSettingsSign(userID string, sign proton.SignExternalMessages) (proton.MailSettings, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.MailSettings, error) {
		return withAcc(b, userID, func(acc *account) (proton.MailSettings, error) {
			acc.mailSettings.sign = sign
			return acc.mailSettings.toMailSettings(), nil
		})
	})
}

func (b *Backend) SetMailSettingsDraftMIMEType(userID string, drafttype rfc822.MIMEType) (proton.MailSettings, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.MailSettings, error) {
		return withAcc(b, userID, func(acc *account) (proton.MailSettings, error) {
			acc.mailSettings.draftMIMEType = drafttype
			return acc.mailSettings.toMailSettings(), nil
		})
	})
}

func (b *Backend) SetMailSettingsPGPScheme(userID string, scheme proton.EncryptionScheme) (proton.MailSettings, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.MailSettings, error) {
		return withAcc(b, userID, func(acc *account) (proton.MailSettings, error) {
			acc.mailSettings.pgpScheme = scheme
			return acc.mailSettings.toMailSettings(), nil
		})
	})
}

func (b *Backend) GetUserSettings(userID string) (proton.UserSettings, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) (proton.UserSettings, error) {
		return withAcc(b, userID, func(acc *account) (proton.UserSettings, error) {
			return acc.userSettings, nil
		})
	})
}

func (b *Backend) SetUserSettingsTelemetry(userID string, telemetry proton.SettingsBool) (proton.UserSettings, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.UserSettings, error) {
		return withAcc(b, userID, func(acc *account) (proton.UserSettings, error) {
			if telemetry != proton.SettingDisabled && telemetry != proton.SettingEnabled {
				return proton.UserSettings{}, errors.New("bad value")
			}
			acc.userSettings.Telemetry = telemetry

			updateID, err := b.newUpdate(&userSettingsUpdate{settings: acc.userSettings})
			if err != nil {
				return acc.userSettings, err
			}

			acc.updateIDs = append(acc.updateIDs, updateID)

			return acc.userSettings, nil
		})
	})
}

func (b *Backend) SetUserSettingsCrashReports(userID string, crashReports proton.SettingsBool) (proton.UserSettings, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.UserSettings, error) {
		return withAcc(b, userID, func(acc *account) (proton.UserSettings, error) {
			if crashReports != proton.SettingDisabled && crashReports != proton.SettingEnabled {
				return proton.UserSettings{}, errors.New("bad value")
			}
			acc.userSettings.CrashReports = crashReports

			updateID, err := b.newUpdate(&userSettingsUpdate{settings: acc.userSettings})
			if err != nil {
				return acc.userSettings, err
			}

			acc.updateIDs = append(acc.updateIDs, updateID)

			return acc.userSettings, nil
		})
	})
}

func (b *Backend) GetAddressID(email string) (string, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) (string, error) {
		return b.getAddressID(email)
	})
}

func (b *unsafeBackend) getAddressID(email string) (string, error) {
	return withAccEmail(b, email, func(acc *account) (string, error) {
		addr, ok := acc.getAddr(email)
		if !ok {
			return "", fmt.Errorf("no such address: %s", email)
		}

		return addr.addrID, nil
	})
}

func (b *Backend) GetAddress(userID, addrID string) (proton.Address, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) (proton.Address, error) {
		return withAcc(b, userID, func(acc *account) (proton.Address, error) {
			if addr, ok := acc.addresses[addrID]; ok {
				return addr.toAddress(), nil
			}

			return proton.Address{}, errors.New("no such address")
		})
	})
}

func (b *Backend) GetAddresses(userID string) ([]proton.Address, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) ([]proton.Address, error) {
		return withAcc(b, userID, func(acc *account) ([]proton.Address, error) {
			return xslices.Map(maps.Values(acc.addresses), func(add *address) proton.Address {
				return add.toAddress()
			}), nil
		})
	})
}

func (b *Backend) EnableAddress(userID, addrID string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			acc.addresses[addrID].status = proton.AddressStatusEnabled

			updateID, err := b.newUpdate(&addressUpdated{addressID: addrID})
			if err != nil {
				return err
			}

			acc.updateIDs = append(acc.updateIDs, updateID)

			return nil
		})
	})
}

func (b *Backend) DisableAddress(userID, addrID string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			acc.addresses[addrID].status = proton.AddressStatusDisabled

			updateID, err := b.newUpdate(&addressUpdated{addressID: addrID})
			if err != nil {
				return err
			}

			acc.updateIDs = append(acc.updateIDs, updateID)

			return nil
		})
	})
}

func (b *Backend) DeleteAddress(userID, addrID string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			if acc.addresses[addrID].status != proton.AddressStatusDisabled {
				return errors.New("address is not disabled")
			}

			delete(acc.addresses, addrID)

			updateID, err := b.newUpdate(&addressDeleted{addressID: addrID})
			if err != nil {
				return err
			}

			acc.updateIDs = append(acc.updateIDs, updateID)

			return nil
		})
	})
}

func (b *Backend) SetAddressOrder(userID string, addrIDs []string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {

			primaryID := acc.primary().addrID

			for i, addrID := range addrIDs {
				if add, ok := acc.addresses[addrID]; ok {
					if add.order != i+1 {
						updateID, err := b.newUpdate(&addressUpdated{addressID: addrID})
						if err != nil {
							return err
						}

						acc.updateIDs = append(acc.updateIDs, updateID)
					}
					add.order = i + 1
				} else {
					return fmt.Errorf("no such address: %s", addrID)
				}
			}

			if primaryID != acc.primary().addrID {
				updateID, err := b.newUpdate(&userInfoUpdate{})
				if err != nil {
					return err
				}

				acc.updateIDs = append(acc.updateIDs, updateID)
			}

			return nil
		})
	})
}

func (b *Backend) HasLabel(userID, labelName string) (string, bool, error) {
	labels, err := b.GetLabels(userID)
	if err != nil {
		return "", false, err
	}

	for _, label := range labels {
		if label.Name == labelName {
			return label.ID, true, nil
		}
	}

	return "", false, nil
}

func (b *Backend) GetLabel(userID, labelID string) (proton.Label, error) {
	labels, err := b.GetLabels(userID)
	if err != nil {
		return proton.Label{}, err
	}

	for _, label := range labels {
		if label.ID == labelID {
			return label, nil
		}
	}

	return proton.Label{}, fmt.Errorf("no such label: %s", labelID)
}

func (b *Backend) GetLabels(userID string, types ...proton.LabelType) ([]proton.Label, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) ([]proton.Label, error) {
		return withAcc(b, userID, func(acc *account) ([]proton.Label, error) {
			return withLabels(b, func(labels map[string]*label) ([]proton.Label, error) {
				res := xslices.Map(acc.labelIDs, func(labelID string) proton.Label {
					return labels[labelID].toLabel(labels)
				})

				for labelName, labelID := range map[string]string{
					"Inbox":     proton.InboxLabel,
					"AllDrafts": proton.AllDraftsLabel,
					"AllSent":   proton.AllSentLabel,
					"Trash":     proton.TrashLabel,
					"Spam":      proton.SpamLabel,
					"All Mail":  proton.AllMailLabel,
					"Archive":   proton.ArchiveLabel,
					"Sent":      proton.SentLabel,
					"Drafts":    proton.DraftsLabel,
					"Outbox":    proton.OutboxLabel,
					"Starred":   proton.StarredLabel,
					"Scheduled": proton.AllScheduledLabel,
				} {
					res = append(res, proton.Label{
						ID:   labelID,
						Name: labelName,
						Path: []string{labelName},
						Type: proton.LabelTypeSystem,
					})
				}

				if len(types) > 0 {
					res = xslices.Filter(res, func(label proton.Label) bool {
						return slices.Contains(types, label.Type)
					})
				}

				return res, nil
			})
		})

	})
}

func (b *Backend) CreateLabel(userID, labelName, parentID string, labelType proton.LabelType) (proton.Label, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.Label, error) {
		return withAcc(b, userID, func(acc *account) (proton.Label, error) {
			return withLabels(b, func(labels map[string]*label) (proton.Label, error) {
				if parentID != "" {
					if labelType != proton.LabelTypeFolder {
						return proton.Label{}, fmt.Errorf("parentID can only be set for folders")
					}

					if _, ok := labels[parentID]; !ok {
						return proton.Label{}, fmt.Errorf("no such parent label: %s", parentID)
					}
				}

				label := newLabel(labelName, parentID, labelType)

				labels[label.labelID] = label

				updateID, err := b.newUpdate(&labelCreated{labelID: label.labelID})
				if err != nil {
					return proton.Label{}, err
				}

				acc.labelIDs = append(acc.labelIDs, label.labelID)
				acc.updateIDs = append(acc.updateIDs, updateID)

				return label.toLabel(labels), nil
			})
		})
	})
}

func (b *Backend) UpdateLabel(userID, labelID, name, parentID string) (proton.Label, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.Label, error) {
		return withAcc(b, userID, func(acc *account) (proton.Label, error) {
			return withLabels(b, func(labels map[string]*label) (proton.Label, error) {
				if parentID != "" {
					if labels[labelID].labelType != proton.LabelTypeFolder {
						return proton.Label{}, fmt.Errorf("parentID can only be set for folders")
					}

					if _, ok := labels[parentID]; !ok {
						return proton.Label{}, fmt.Errorf("no such parent label: %s", parentID)
					}
				}

				labels[labelID].name = name
				labels[labelID].parentID = parentID

				updateID, err := b.newUpdate(&labelUpdated{labelID: labelID})
				if err != nil {
					return proton.Label{}, err
				}

				acc.updateIDs = append(acc.updateIDs, updateID)

				return labels[labelID].toLabel(labels), nil
			})
		})
	})
}

func (b *Backend) DeleteLabel(userID, labelID string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			return b.withLabels(func(labels map[string]*label) error {
				if _, ok := labels[labelID]; !ok {
					return errors.New("label not found")
				}

				for _, labelID := range getLabelIDsToDelete(labelID, labels) {
					delete(labels, labelID)

					updateID, err := b.newUpdate(&labelDeleted{labelID: labelID})
					if err != nil {
						return err
					}

					acc.labelIDs = xslices.Filter(acc.labelIDs, func(otherID string) bool { return otherID != labelID })
					acc.updateIDs = append(acc.updateIDs, updateID)
				}

				return nil
			})
		})
	})
}

func (b *Backend) CountMessages(userID string) (int, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) (int, error) {
		return withAcc(b, userID, func(acc *account) (int, error) {
			return len(acc.messageIDs), nil
		})
	})
}

func (b *Backend) GetMessageIDs(userID string, afterID string, limit int) ([]string, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) ([]string, error) {
		return withAcc(b, userID, func(acc *account) ([]string, error) {
			if len(acc.messageIDs) == 0 {
				return nil, nil
			}

			var lo, hi int

			if afterID == "" {
				lo = 0
			} else {
				lo = slices.Index(acc.messageIDs, afterID) + 1
			}

			if limit == 0 {
				hi = len(acc.messageIDs)
			} else {
				hi = lo + limit

				if hi > len(acc.messageIDs) {
					hi = len(acc.messageIDs)
				}
			}

			return acc.messageIDs[lo:hi], nil
		})
	})
}

func (b *Backend) GetMessages(userID string, page, pageSize int, filter proton.MessageFilter) ([]proton.MessageMetadata, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) ([]proton.MessageMetadata, error) {
		return withAcc(b, userID, func(acc *account) ([]proton.MessageMetadata, error) {
			return withMessages(b, func(messages map[string]*message) ([]proton.MessageMetadata, error) {
				metadata, err := withAtts(b, func(atts map[string]*attachment) ([]proton.MessageMetadata, error) {
					return xslices.Map(acc.messageIDs, func(messageID string) proton.MessageMetadata {
						return messages[messageID].toMetadata(b.attData, atts)
					}), nil
				})
				if err != nil {
					return nil, err
				}

				if filter.Desc {
					xslices.Reverse(metadata)
				}

				// Note that this not a perfect replacement as we don't handle the case where this message could have been
				// deleted in between this metadata request. The backend has the information stored differently and can
				// resolve these gaps.
				if filter.EndID != "" {
					index := xslices.IndexFunc(metadata, func(metadata proton.MessageMetadata) bool {
						return metadata.ID == filter.EndID
					})

					if index >= 0 {
						metadata = metadata[index:]
					}
				}

				metadata = xslices.Filter(metadata, func(metadata proton.MessageMetadata) bool {
					if len(filter.ID) > 0 {
						if !slices.Contains(filter.ID, metadata.ID) {
							return false
						}
					}

					if filter.Subject != "" {
						if !strings.Contains(metadata.Subject, filter.Subject) {
							return false
						}
					}

					if filter.AddressID != "" {
						if filter.AddressID != metadata.AddressID {
							return false
						}
					}

					if filter.ExternalID != "" {
						if filter.ExternalID != metadata.ExternalID {
							return false
						}
					}

					if filter.LabelID != "" {
						if !slices.Contains(metadata.LabelIDs, filter.LabelID) {
							return false
						}
					}

					return true
				})

				pages := xslices.Chunk(metadata, pageSize)
				if page >= len(pages) {
					return nil, nil
				}

				return pages[page], nil
			})
		})
	})
}

func (b *Backend) GetMessage(userID, messageID string) (proton.Message, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) (proton.Message, error) {
		return withAcc(b, userID, func(acc *account) (proton.Message, error) {
			return withMessages(b, func(messages map[string]*message) (proton.Message, error) {
				return withAtts(b, func(atts map[string]*attachment) (proton.Message, error) {
					message, ok := messages[messageID]
					if !ok {
						return proton.Message{}, errors.New("no such message")
					}

					return message.toMessage(b.attData, atts), nil
				})
			})
		})
	})
}

func (b *Backend) SetMessagesRead(userID string, read bool, messageIDs ...string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			return b.withMessages(func(messages map[string]*message) error {
				for _, messageID := range messageIDs {
					messages[messageID].unread = !read

					updateID, err := b.newUpdate(&messageUpdated{messageID: messageID})
					if err != nil {
						return err
					}

					acc.updateIDs = append(acc.updateIDs, updateID)
				}

				return nil
			})
		})
	})
}

func (b *Backend) SetMessagesForwarded(userID string, forwarded bool, messageIDs ...string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			return b.withMessages(func(messages map[string]*message) error {
				for _, messageID := range messageIDs {
					if forwarded {
						messages[messageID].flags |= proton.MessageFlagForwarded
					} else {
						messages[messageID].flags &= ^proton.MessageFlagForwarded
					}

					updateID, err := b.newUpdate(&messageUpdated{messageID: messageID})
					if err != nil {
						return err
					}

					acc.updateIDs = append(acc.updateIDs, updateID)
				}

				return nil
			})
		})
	})
}

func (b *Backend) LabelMessages(userID, labelID string, messageIDs ...string) error {
	return b.labelMessages(userID, labelID, true, messageIDs...)
}

func (b *Backend) LabelMessagesNoEvents(userID, labelID string, messageIDs ...string) error {
	return b.labelMessages(userID, labelID, false, messageIDs...)
}

func (b *Backend) labelMessages(userID, labelID string, doEvents bool, messageIDs ...string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		if labelID == proton.AllMailLabel || labelID == proton.AllDraftsLabel || labelID == proton.AllSentLabel {
			return fmt.Errorf("not allowed")
		}

		return b.withAcc(userID, func(acc *account) error {
			return b.withMessages(func(messages map[string]*message) error {
				return b.withLabels(func(labels map[string]*label) error {
					for _, messageID := range messageIDs {
						message, ok := messages[messageID]
						if !ok {
							continue
						}

						message.addLabel(labelID, labels)

						if doEvents {
							updateID, err := b.newUpdate(&messageUpdated{messageID: messageID})
							if err != nil {
								return err
							}

							acc.updateIDs = append(acc.updateIDs, updateID)
						}
					}

					return nil
				})
			})
		})
	})
}

func (b *Backend) UnlabelMessages(userID, labelID string, messageIDs ...string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		if labelID == proton.AllMailLabel || labelID == proton.AllDraftsLabel || labelID == proton.AllSentLabel {
			return fmt.Errorf("not allowed")
		}

		return b.withAcc(userID, func(acc *account) error {
			return b.withMessages(func(messages map[string]*message) error {
				return b.withLabels(func(labels map[string]*label) error {
					for _, messageID := range messageIDs {
						messages[messageID].remLabel(labelID, labels)

						updateID, err := b.newUpdate(&messageUpdated{messageID: messageID})
						if err != nil {
							return err
						}

						acc.updateIDs = append(acc.updateIDs, updateID)
					}

					return nil
				})
			})
		})
	})
}

func (b *Backend) DeleteMessage(userID, messageID string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			return b.withMessages(func(messages map[string]*message) error {
				message, ok := messages[messageID]
				if !ok {
					return errors.New("no such message")
				}

				for _, attID := range message.attIDs {
					if xslices.CountFunc(maps.Values(b.attachments), func(att *attachment) bool {
						return att.attDataID == b.attachments[attID].attDataID
					}) == 1 {
						delete(b.attData, b.attachments[attID].attDataID)
					}

					delete(b.attachments, attID)
				}

				delete(b.messages, messageID)

				updateID, err := b.newUpdate(&messageDeleted{messageID: messageID})
				if err != nil {
					return err
				}

				acc.messageIDs = xslices.Filter(acc.messageIDs, func(otherID string) bool { return otherID != messageID })
				acc.updateIDs = append(acc.updateIDs, updateID)

				return nil
			})
		})
	})
}

func (b *Backend) CreateDraft(userID, addrID string, draft proton.DraftTemplate, parentID string, action proton.CreateDraftAction) (proton.Message, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.Message, error) {
		return withAcc(b, userID, func(acc *account) (proton.Message, error) {
			return withMessages(b, func(messages map[string]*message) (proton.Message, error) {
				return withLabels(b, func(labels map[string]*label) (proton.Message, error) {
					// Convert the parentID into externalRef.\
					var parentRef string
					var internalParentID string
					if parentID != "" {
						parentMsg, ok := messages[parentID]
						if ok {
							parentRef = "<" + parentMsg.externalID + ">"
							internalParentID = parentID
						}
					}
					msg := newMessageFromTemplate(addrID, draft, parentRef, internalParentID, action)
					// Drafts automatically get the sysLabel "Drafts".
					msg.addLabel(proton.DraftsLabel, labels)

					messages[msg.messageID] = msg

					updateID, err := b.newUpdate(&messageCreated{messageID: msg.messageID})
					if err != nil {
						return proton.Message{}, err
					}

					acc.messageIDs = append(acc.messageIDs, msg.messageID)
					acc.updateIDs = append(acc.updateIDs, updateID)

					return msg.toMessage(nil, nil), nil
				})
			})
		})

	})
}

func (b *Backend) UpdateDraft(userID, draftID string, changes proton.DraftTemplate) (proton.Message, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.Message, error) {
		if changes.Sender == nil {
			return proton.Message{}, errors.New("the Sender is required")
		}

		if changes.Sender.Address == "" {
			return proton.Message{}, errors.New("the Address is required")
		}

		if changes.MIMEType != rfc822.TextPlain && changes.MIMEType != rfc822.TextHTML {
			return proton.Message{}, errors.New("the MIMEType must be text/plain or text/html")
		}

		return withAcc(b, userID, func(acc *account) (proton.Message, error) {
			return withMessages(b, func(messages map[string]*message) (proton.Message, error) {
				return withAtts(b, func(atts map[string]*attachment) (proton.Message, error) {
					if _, ok := messages[draftID]; !ok {
						return proton.Message{}, fmt.Errorf("message %q not found", draftID)
					}

					messages[draftID].applyChanges(changes)

					updateID, err := b.newUpdate(&messageUpdated{messageID: draftID})
					if err != nil {
						return proton.Message{}, err
					}

					acc.updateIDs = append(acc.updateIDs, updateID)

					return messages[draftID].toMessage(b.attData, atts), nil
				})
			})
		})
	})
}

func (b *Backend) SendMessage(userID, messageID string, packages []*proton.MessagePackage) (proton.Message, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.Message, error) {
		return withAcc(b, userID, func(acc *account) (proton.Message, error) {
			return withMessages(b, func(messages map[string]*message) (proton.Message, error) {
				return withLabels(b, func(labels map[string]*label) (proton.Message, error) {
					return withAtts(b, func(atts map[string]*attachment) (proton.Message, error) {
						msg := messages[messageID]
						msg.flags |= proton.MessageFlagSent
						msg.addLabel(proton.SentLabel, labels)

						if parent, ok := messages[msg.internalParentID]; ok {
							switch msg.draftAction {
							case proton.ReplyAction:
								parent.flags |= proton.MessageFlagReplied
							case proton.ReplyAllAction:
								parent.flags |= proton.MessageFlagRepliedAll
							case proton.ForwardAction:
								parent.flags |= proton.MessageFlagForwarded
							}

							updateID, err := b.newUpdate(&messageUpdated{messageID: msg.internalParentID})
							if err != nil {
								return proton.Message{}, err
							}

							acc.updateIDs = append(acc.updateIDs, updateID)
						}

						updateID, err := b.newUpdate(&messageUpdated{messageID: messageID})
						if err != nil {
							return proton.Message{}, err
						}

						acc.updateIDs = append(acc.updateIDs, updateID)

						for _, pkg := range packages {
							bodyData, err := base64.StdEncoding.DecodeString(pkg.Body)
							if err != nil {
								return proton.Message{}, err
							}

							for email, recipient := range pkg.Addresses {
								if recipient.Type != proton.InternalScheme {
									continue
								}

								if err := b.withAccEmail(email, func(acc *account) error {
									bodyKey, err := base64.StdEncoding.DecodeString(recipient.BodyKeyPacket)
									if err != nil {
										return err
									}

									armBody, err := crypto.NewPGPSplitMessage(bodyKey, bodyData).GetPGPMessage().GetArmored()
									if err != nil {
										return err
									}

									addrID, err := b.getAddressID(email)
									if err != nil {
										return err
									}

									newMsg := newMessageFromSent(addrID, armBody, msg)
									newMsg.flags |= proton.MessageFlagReceived
									newMsg.addLabel(proton.InboxLabel, labels)
									newMsg.unread = true
									messages[newMsg.messageID] = newMsg

									for _, attID := range msg.attIDs {
										attKey, err := base64.StdEncoding.DecodeString(recipient.AttachmentKeyPackets[attID])
										if err != nil {
											return err
										}

										att := newAttachment(
											atts[attID].filename,
											atts[attID].mimeType,
											atts[attID].disposition,
											atts[attID].contentID,
											attKey,
											atts[attID].attDataID,
											atts[attID].armSig,
										)
										atts[att.attachID] = att
										messages[newMsg.messageID].attIDs = append(messages[newMsg.messageID].attIDs, att.attachID)
									}
									// Sort Message attachments
									messages[newMsg.messageID].attIDs = sortAttachment(atts, messages[newMsg.messageID].attIDs)
									msg.attIDs = sortAttachment(atts, msg.attIDs)

									// Send the update event
									updateID, err := b.newUpdate(&messageCreated{messageID: newMsg.messageID})
									if err != nil {
										return err
									}

									acc.messageIDs = append(acc.messageIDs, newMsg.messageID)
									acc.updateIDs = append(acc.updateIDs, updateID)

									return nil
								}); err != nil {
									return proton.Message{}, err
								}
							}
						}

						return msg.toMessage(b.attData, atts), nil
					})
				})
			})
		})
	})
}

func sortAttachment(atts map[string]*attachment, attIDs []string) []string {
	// collect attachment with contentID
	attContentId := make(map[string]string, len(attIDs))
	for _, attID := range attIDs {
		if atts[attID].contentID != "" {
			attContentId[atts[attID].contentID] = attID
		}
	}
	// sort contentID
	contentIDs := make([]string, 0, len(attContentId))
	for k := range attContentId {
		contentIDs = append(contentIDs, k)
	}
	sort.Strings(contentIDs)
	// set contentID attachment first
	attachments := make([]string, 0, len(attIDs))
	for _, id := range contentIDs {
		attachments = append(attachments, attContentId[id])
	}

	// treat the rest by filename
	attNames := make(map[string]string, len(attIDs))
	for _, attID := range attIDs {
		if atts[attID].contentID == "" {
			attNames[atts[attID].filename] = attID
		}
	}
	// sort name
	names := make([]string, 0, len(attNames))
	for k := range attNames {
		names = append(names, k)
	}
	sort.Strings(names)
	// Append by alphabetical order
	for _, name := range names {
		attachments = append(attachments, attNames[name])
	}
	return attachments
}

func (b *Backend) CreateAttachment(
	userID string,
	messageID string,
	filename string,
	mimeType rfc822.MIMEType,
	disposition proton.Disposition,
	contentID string,
	keyPackets, dataPacket []byte,
	armSig string,
) (proton.Attachment, error) {
	if disposition != proton.InlineDisposition && disposition != proton.AttachmentDisposition {
		return proton.Attachment{}, errors.New("The Disposition only allows 'attachment', or 'inline'")
	}

	if disposition == proton.InlineDisposition && contentID == "" {
		return proton.Attachment{}, errors.New("The 'inline' Disposition is only allowed with Content ID")
	}

	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.Attachment, error) {
		return withAcc(b, userID, func(acc *account) (proton.Attachment, error) {
			return withMessages(b, func(messages map[string]*message) (proton.Attachment, error) {
				return withAtts(b, func(atts map[string]*attachment) (proton.Attachment, error) {
					att := newAttachment(
						filename,
						mimeType,
						disposition,
						contentID,
						keyPackets,
						b.createAttData(dataPacket),
						armSig,
					)

					atts[att.attachID] = att

					// append attachment
					messages[messageID].attIDs = append(messages[messageID].attIDs, att.attachID)
					// sort the attachment list
					messages[messageID].attIDs = sortAttachment(atts, messages[messageID].attIDs)

					updateID, err := b.newUpdate(&messageUpdated{messageID: messageID})
					if err != nil {
						return proton.Attachment{}, err
					}

					acc.updateIDs = append(acc.updateIDs, updateID)

					return att.toAttachment(), nil
				})
			})
		})
	})
}

func (b *Backend) GetAttachment(attachID string) ([]byte, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) ([]byte, error) {
		return withAtts(b, func(atts map[string]*attachment) ([]byte, error) {
			att, ok := atts[attachID]
			if !ok {
				return nil, fmt.Errorf("no such attachment: %s", attachID)
			}

			return b.attData[att.attDataID], nil
		})
	})
}

func (b *Backend) GetLatestEventID(userID string) (string, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) (string, error) {
		return withAcc(b, userID, func(acc *account) (string, error) {
			return acc.updateIDs[len(acc.updateIDs)-1].String(), nil
		})
	})
}

func getLastUpdateIndex(total, first, max int) int {
	if max <= 0 || first+max > total {
		return total
	}

	return first + max
}

func (b *Backend) GetEvent(userID, rawEventID string) (event proton.Event, more bool, err error) {
	var eventID ID

	if err := eventID.FromString(rawEventID); err != nil {
		return proton.Event{}, false, fmt.Errorf("invalid event ID: %s", rawEventID)
	}

	more = false

	event, err = readBackendRetErr(b, func(b *unsafeBackend) (proton.Event, error) {
		return withAcc(b, userID, func(acc *account) (proton.Event, error) {
			return withMessages(b, func(messages map[string]*message) (proton.Event, error) {
				return withLabels(b, func(labels map[string]*label) (proton.Event, error) {
					return withAtts(b, func(attachments map[string]*attachment) (proton.Event, error) {
						firstUpdate := xslices.Index(acc.updateIDs, eventID) + 1
						lastUpdate := getLastUpdateIndex(len(acc.updateIDs), firstUpdate, b.maxUpdatesPerEvent)

						updates, err := withUpdates(b, func(updates map[ID]update) ([]update, error) {
							return merge(xslices.Map(acc.updateIDs[firstUpdate:lastUpdate], func(updateID ID) update {
								return updates[updateID]
							})), nil
						})
						if err != nil {
							return proton.Event{}, fmt.Errorf("failed to merge updates: %w", err)
						}

						more = lastUpdate != len(acc.updateIDs)

						return buildEvent(updates, acc.addresses, messages, labels, acc.updateIDs[lastUpdate-1].String(), b.attData, attachments, acc.toUser()), nil
					})
				})
			})
		})
	})

	if err != nil {
		return proton.Event{}, false, err
	}

	return event, more, nil
}

func (b *Backend) GetPublicKeys(email string) ([]proton.PublicKey, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) ([]proton.PublicKey, error) {
		return withAccEmail(b, email, func(acc *account) ([]proton.PublicKey, error) {
			var keys []proton.PublicKey

			for _, addr := range acc.addresses {
				if addr.email == email {
					for _, key := range addr.keys {
						pubKey, err := key.getPubKey()
						if err != nil {
							return nil, err
						}

						armKey, err := pubKey.GetArmoredPublicKey()
						if err != nil {
							return nil, err
						}

						keys = append(keys, proton.PublicKey{
							Flags:     proton.KeyStateTrusted | proton.KeyStateActive,
							PublicKey: armKey,
						})
					}
				}
			}

			return keys, nil
		})
	})
}

func getLabelIDsToDelete(labelID string, labels map[string]*label) []string {
	labelIDs := []string{labelID}

	for _, label := range labels {
		if label.parentID == labelID {
			labelIDs = append(labelIDs, getLabelIDsToDelete(label.labelID, labels)...)
		}
	}

	return labelIDs
}

func (b *Backend) AddAddressCreatedUpdate(userID, addrID string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			updateID, err := b.newUpdate(&addressCreated{addressID: addrID})
			if err != nil {
				return err
			}

			acc.updateIDs = append(acc.updateIDs, updateID)

			return nil
		})
	})
}

func (b *Backend) AddLabelCreatedUpdate(userID, labelID string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			updateID, err := b.newUpdate(&labelCreated{labelID: labelID})
			if err != nil {
				return err
			}

			acc.updateIDs = append(acc.updateIDs, updateID)

			return nil
		})
	})
}

func (b *Backend) AddMessageCreatedUpdate(userID, messageID string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			updateID, err := b.newUpdate(&messageCreated{messageID: messageID})
			if err != nil {
				return err
			}

			acc.updateIDs = append(acc.updateIDs, updateID)

			return nil
		})
	})
}

func (b *Backend) GetMessageGroupCount(userID string) ([]proton.MessageGroupCount, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) ([]proton.MessageGroupCount, error) {
		var result []proton.MessageGroupCount
		err := b.withAcc(userID, func(acc *account) error {
			return b.withMessages(func(m map[string]*message) error {
				type stats struct {
					total  int
					unread int
				}

				labelStats := make(map[string]stats)

				for _, msg := range m {
					for _, lbl := range msg.getLabelIDs() {
						v, ok := labelStats[lbl]
						if !ok {
							v = stats{}
						}
						v.total++
						if msg.unread {
							v.unread++
						}

						labelStats[lbl] = v
					}
				}

				result = make([]proton.MessageGroupCount, 0, len(labelStats))

				for lbl, stats := range labelStats {
					result = append(result, proton.MessageGroupCount{
						LabelID: lbl,
						Total:   stats.total,
						Unread:  stats.unread,
					})
				}

				return nil
			})
		})

		return result, err
	})
}

func (b *Backend) GetUserContact(userID, contactID string) (proton.Contact, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) (proton.Contact, error) {
		return withAcc(b, userID, func(acc *account) (proton.Contact, error) {
			contact, ok := acc.contacts[contactID]
			if ok {
				return *contact, nil
			}
			return proton.Contact{}, errors.New("No such contact with ID" + contactID)
		})
	})
}

func (b *Backend) GetUserContacts(userID string, page int, pageSize int) (int, []proton.Contact, error) {
	var total int
	contacts, err := readBackendRetErr(b, func(b *unsafeBackend) ([]proton.Contact, error) {
		return withAcc(b, userID, func(acc *account) ([]proton.Contact, error) {
			total = len(acc.contacts)
			values := maps.Values(acc.contacts)
			slices.SortFunc(values, func(i, j *proton.Contact) bool {
				return strings.Compare(i.ID, j.ID) < 0
			})
			return xslices.Map(xslices.Chunk(values, pageSize)[page], func(c *proton.Contact) proton.Contact {
				return *c
			}), nil
		})
	})

	return total, contacts, err
}

func (b *Backend) GetUserContactEmails(userID, email string, page int, pageSize int) (int, []proton.ContactEmail, error) {
	var total int

	emails, err := readBackendRetErr(b, func(b *unsafeBackend) ([]proton.ContactEmail, error) {
		return withAcc(b, userID, func(acc *account) ([]proton.ContactEmail, error) {
			var contacts []proton.ContactEmail
			for _, contact := range acc.contacts {
				for _, contactEmail := range contact.ContactEmails {
					if email == "" || contactEmail.Email == email {
						contacts = append(contacts, contactEmail)
					}
				}
			}

			total = len(contacts)

			if total < pageSize {
				return contacts, nil
			}

			slices.SortFunc(contacts, func(a, b proton.ContactEmail) bool {
				return strings.Compare(a.ID, b.ID) < 0
			})

			return xslices.Chunk(contacts, pageSize)[page], nil
		})
	})

	return total, emails, err
}

func (b *Backend) AddUserContact(userID string, contact proton.Contact) (proton.Contact, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.Contact, error) {
		return withAcc(b, userID, func(acc *account) (proton.Contact, error) {
			acc.contacts[contact.ID] = &contact
			return *acc.contacts[contact.ID], nil
		})
	})
}

func (b *Backend) UpdateUserContact(userID, contactID string, cards proton.Cards) (proton.Contact, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.Contact, error) {
		return withAcc(b, userID, func(acc *account) (proton.Contact, error) {
			acc.contacts[contactID].Cards = cards
			return *acc.contacts[contactID], nil
		})
	})
}

func (b *Backend) GenerateContactID(userID string) (string, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (string, error) {
		return withAcc(b, userID, func(acc *account) (string, error) {
			acc.contactCounter++
			return strconv.Itoa(acc.contactCounter), nil
		})
	})
}

func (b *Backend) GetFeatureFlags() []proton.FeatureToggle {
	return readBackendRet(b, func(b *unsafeBackend) []proton.FeatureToggle {
		return b.featureFlags
	})
}

func (b *Backend) PushFeatureFlag(flagName string) {
	writeBackend(b, func(b *unsafeBackend) {
		b.featureFlags = append(b.featureFlags, proton.FeatureToggle{
			Name:    flagName,
			Enabled: true,
		})
	})
}

func (b *Backend) DeleteFeatureFlags() {
	writeBackend(b, func(b *unsafeBackend) {
		b.featureFlags = []proton.FeatureToggle{}
	})
}

func buildEvent(
	updates []update,
	addresses map[string]*address,
	messages map[string]*message,
	labels map[string]*label,
	eventID string,
	attachmentData map[string][]byte,
	attachments map[string]*attachment,
	user proton.User,
) proton.Event {
	event := proton.Event{EventID: eventID}

	for _, update := range updates {
		switch update := update.(type) {
		case *userRefreshed:
			event.Refresh = update.refresh

		case *messageCreated:
			event.Messages = append(event.Messages, proton.MessageEvent{
				EventItem: proton.EventItem{
					ID:     update.messageID,
					Action: proton.EventCreate,
				},

				Message: messages[update.messageID].toMetadata(attachmentData, attachments),
			})

		case *messageUpdated:
			event.Messages = append(event.Messages, proton.MessageEvent{
				EventItem: proton.EventItem{
					ID:     update.messageID,
					Action: proton.EventUpdate,
				},

				Message: messages[update.messageID].toMetadata(attachmentData, attachments),
			})

		case *messageDeleted:
			event.Messages = append(event.Messages, proton.MessageEvent{
				EventItem: proton.EventItem{
					ID:     update.messageID,
					Action: proton.EventDelete,
				},
			})

		case *labelCreated:
			event.Labels = append(event.Labels, proton.LabelEvent{
				EventItem: proton.EventItem{
					ID:     update.labelID,
					Action: proton.EventCreate,
				},

				Label: labels[update.labelID].toLabel(labels),
			})

		case *labelUpdated:
			event.Labels = append(event.Labels, proton.LabelEvent{
				EventItem: proton.EventItem{
					ID:     update.labelID,
					Action: proton.EventUpdate,
				},

				Label: labels[update.labelID].toLabel(labels),
			})

		case *labelDeleted:
			event.Labels = append(event.Labels, proton.LabelEvent{
				EventItem: proton.EventItem{
					ID:     update.labelID,
					Action: proton.EventDelete,
				},
			})

		case *addressCreated:
			event.Addresses = append(event.Addresses, proton.AddressEvent{
				EventItem: proton.EventItem{
					ID:     update.addressID,
					Action: proton.EventCreate,
				},

				Address: addresses[update.addressID].toAddress(),
			})

		case *addressUpdated:
			event.Addresses = append(event.Addresses, proton.AddressEvent{
				EventItem: proton.EventItem{
					ID:     update.addressID,
					Action: proton.EventUpdate,
				},

				Address: addresses[update.addressID].toAddress(),
			})

		case *addressDeleted:
			event.Addresses = append(event.Addresses, proton.AddressEvent{
				EventItem: proton.EventItem{
					ID:     update.addressID,
					Action: proton.EventDelete,
				},
			})

		case *userSettingsUpdate:
			event.UserSettings = &proton.UserSettings{
				Telemetry:    update.settings.Telemetry,
				CrashReports: update.settings.CrashReports,
			}

		case *userInfoUpdate:
			event.User = &user
		}
	}

	return event
}
