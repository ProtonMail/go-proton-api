package proton

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/go-resty/resty/v2"
)

// APIHVDetails contains information related to the human verification requests.
type APIHVDetails struct {
	Methods []string `json:"HumanVerificationMethods"`
	Token   string   `json:"HumanVerificationToken"`
}

func addHVToRequest(req *resty.Request, hv *APIHVDetails) *resty.Request {
	if hv == nil {
		return req
	}

	return req.SetHeader(hvPMTokenHeaderField, hv.Token).SetHeader(hvPMTokenType, strings.Join(hv.Methods, ","))
}

var ErrAPIErrIsNotHVErr = errors.New("not HV error")

func (err APIError) GetHVDetails() (*APIHVDetails, error) {
	if !err.IsHVError() {
		return nil, ErrAPIErrIsNotHVErr
	}

	r := new(APIHVDetails)

	if err := json.Unmarshal(err.Details, &r); err != nil {
		return nil, err
	}

	return r, nil
}
