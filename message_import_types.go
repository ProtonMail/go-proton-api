package proton

import (
	"encoding/json"

	"github.com/ProtonMail/gluon/rfc822"
	"github.com/go-resty/resty/v2"
)

type ImportReq struct {
	Metadata ImportMetadata
	Message  []byte
}

type namedImportReq struct {
	ImportReq

	Name string
}

type ImportMetadata struct {
	AddressID string
	LabelIDs  []string
	Unread    Bool
	Flags     MessageFlag
}

type ImportRes struct {
	Response  APIError
	MessageID string
}

func buildImportReqFields(req []namedImportReq) ([]*resty.MultipartField, error) {
	var fields []*resty.MultipartField

	metadata := make(map[string]ImportMetadata, len(req))

	for _, req := range req {
		metadata[req.Name] = req.Metadata

		fields = append(fields, &resty.MultipartField{
			Param:       req.Name,
			FileName:    req.Name + ".eml",
			ContentType: string(rfc822.MessageRFC822),
			Stream:      resty.NewByteMultipartStream(append(req.Message, "\r\n"...)),
		})
	}

	b, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	fields = append(fields, &resty.MultipartField{
		Param:       "Metadata",
		ContentType: "application/json",
		Stream:      resty.NewByteMultipartStream(b),
	})

	return fields, nil
}
