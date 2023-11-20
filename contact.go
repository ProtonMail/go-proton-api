package proton

import (
	"context"
	"strconv"

	"github.com/go-resty/resty/v2"
)

func (c *Client) GetContact(ctx context.Context, contactID string) (Contact, error) {
	var res struct {
		Contact Contact
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/contacts/v4/" + contactID)
	}); err != nil {
		return Contact{}, err
	}

	return res.Contact, nil
}

func (c *Client) CountContacts(ctx context.Context) (int, error) {
	var res struct {
		Total int
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/contacts/v4")
	}); err != nil {
		return 0, err
	}

	return res.Total, nil
}

func (c *Client) CountContactEmails(ctx context.Context, email string) (int, error) {
	var res struct {
		Total int
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).SetQueryParam("Email", email).Get("/contacts/v4/emails")
	}); err != nil {
		return 0, err
	}

	return res.Total, nil
}

func (c *Client) GetContacts(ctx context.Context, page, pageSize int) ([]Contact, error) {
	_, contacts, err := c.getContactsImpl(ctx, page, pageSize)

	return contacts, err
}

func (c *Client) getContactsImpl(ctx context.Context, page, pageSize int) (int, []Contact, error) {
	var res struct {
		Contacts []Contact
		Total    int
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetQueryParams(map[string]string{
			"Page":     strconv.Itoa(page),
			"PageSize": strconv.Itoa(pageSize),
		}).SetResult(&res).Get("/contacts/v4")
	}); err != nil {
		return 0, nil, err
	}

	return res.Total, res.Contacts, nil
}

func (c *Client) GetAllContacts(ctx context.Context) ([]Contact, error) {
	return c.GetAllContactsPaged(ctx, maxPageSize)
}

func (c *Client) GetAllContactsPaged(ctx context.Context, pageSize int) ([]Contact, error) {
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	total, firstBatch, err := c.getContactsImpl(ctx, 0, pageSize)
	if err != nil {
		return nil, err
	}

	if total <= pageSize {
		return firstBatch, nil
	}

	remainingPages := (total / pageSize) + 1

	for i := 1; i < remainingPages; i++ {
		_, batch, err := c.getContactsImpl(ctx, i, pageSize)
		if err != nil {
			return nil, err
		}

		firstBatch = append(firstBatch, batch...)
	}

	return firstBatch, err
}

func (c *Client) GetContactEmails(ctx context.Context, email string, page, pageSize int) ([]ContactEmail, error) {
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	_, contacts, err := c.getContactEmailsImpl(ctx, email, page, pageSize)

	return contacts, err
}

func (c *Client) getContactEmailsImpl(ctx context.Context, email string, page, pageSize int) (int, []ContactEmail, error) {
	var res struct {
		ContactEmails []ContactEmail
		Total         int
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetQueryParams(map[string]string{
			"Page":     strconv.Itoa(page),
			"PageSize": strconv.Itoa(pageSize),
			"Email":    email,
		}).SetResult(&res).Get("/contacts/v4/emails")
	}); err != nil {
		return 0, nil, err
	}

	return res.Total, res.ContactEmails, nil
}

func (c *Client) GetAllContactEmails(ctx context.Context, email string) ([]ContactEmail, error) {
	return c.GetAllContactEmailsPaged(ctx, email, maxPageSize)
}

func (c *Client) GetAllContactEmailsPaged(ctx context.Context, email string, pageSize int) ([]ContactEmail, error) {
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	total, firstBatch, err := c.getContactEmailsImpl(ctx, email, 0, pageSize)
	if err != nil {
		return nil, err
	}

	if total <= pageSize {
		return firstBatch, nil
	}

	remainingPages := (total / pageSize) + 1

	for i := 1; i < remainingPages; i++ {
		_, batch, err := c.getContactEmailsImpl(ctx, email, i, pageSize)
		if err != nil {
			return nil, err
		}

		firstBatch = append(firstBatch, batch...)
	}

	return firstBatch, err
}

func (c *Client) CreateContacts(ctx context.Context, req CreateContactsReq) ([]CreateContactsRes, error) {
	var res struct {
		Responses []CreateContactsRes
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetBody(req).SetResult(&res).Post("/contacts/v4")
	}); err != nil {
		return nil, err
	}

	return res.Responses, nil
}

func (c *Client) UpdateContact(ctx context.Context, contactID string, req UpdateContactReq) (Contact, error) {
	var res struct {
		Contact Contact
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetBody(req).SetResult(&res).Put("/contacts/v4/" + contactID)
	}); err != nil {
		return Contact{}, err
	}

	return res.Contact, nil
}

func (c *Client) DeleteContacts(ctx context.Context, req DeleteContactsReq) error {
	return c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetBody(req).Put("/contacts/v4/delete")
	})
}
