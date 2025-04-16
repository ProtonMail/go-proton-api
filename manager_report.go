package proton

import (
	"bytes"
	"context"

	"github.com/go-resty/resty/v2"
)

func (m *Manager) ReportBug(ctx context.Context, req ReportBugReq, atts ...ReportBugAttachment) (ReportBugRes, error) {
	r := m.r(ctx).SetMultipartFormData(req.toFormData())

	for _, att := range atts {
		r = r.SetMultipartField(att.Name, att.Filename, string(att.MIMEType), bytes.NewReader(att.Body))
	}
	var res ReportBugRes

	if resp, err := r.SetResult(&res).Post("/core/v4/reports/bug"); err != nil {
		if resp != nil {
			return ReportBugRes{}, &resty.ResponseError{Response: resp, Err: err}
		}
		return ReportBugRes{}, err
	}

	return res, nil
}

func (m *Manager) ReportBugAttachement(ctx context.Context, req ReportBugAttachmentReq, atts ...ReportBugAttachment) error {
	r := m.r(ctx).SetMultipartFormData(req.toFormData())

	for _, att := range atts {
		r = r.SetMultipartField(att.Name, att.Filename, string(att.MIMEType), bytes.NewReader(att.Body))
	}

	if _, err := r.Post("/core/v4/reports/bug/attachments"); err != nil {
		return err
	}

	return nil
}
