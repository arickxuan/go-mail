package pop3

import (
	"fmt"
	"io"
	"mail0/store"
	"strconv"
	"time"

	pop3 "github.com/knadh/go-pop3"
)

type Client struct {
	account *store.Account
}

func New(a *store.Account) *Client {
	return &Client{account: a}
}

func (c *Client) Check() error {
	conn, err := c.dial()
	if err != nil {
		return err
	}
	defer conn.Quit()
	return nil
}

func (c *Client) dial() (*pop3.Conn, error) {
	a := c.account
	server := a.POP3Server
	port := a.POP3Port
	tlsMode := a.POP3TLS

	if server == "" {
		server, port, tlsMode = store.POP3ProviderConfig(a.ProviderType)
	}

	p := pop3.New(pop3.Opt{
		Host:       server,
		Port:       port,
		TLSEnabled: tlsMode,
	})

	conn, err := p.NewConn()
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	if err := conn.Auth(a.Email, a.Password); err != nil {
		conn.Quit()
		return nil, fmt.Errorf("auth: %w", err)
	}

	return conn, nil
}

func (c *Client) ListInbox(page, pageSize int) (*store.EmailPage, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Quit()

	count, _, err := conn.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}

	total := count
	if total == 0 {
		return &store.EmailPage{
			AccountID: c.account.ID,
			Folder:    "inbox",
			Page:      page,
			Total:     0,
			Emails:    []store.EmailSummary{},
		}, nil
	}

	start := total - (page-1)*pageSize
	if start < 1 {
		start = 1
	}
	end := start - pageSize + 1
	if end < 1 {
		end = 1
	}

	emails := make([]store.EmailSummary, 0)
	for id := start; id >= end; id-- {
		m, err := conn.Retr(id)
		if err != nil {
			continue
		}
		h := m.Header
		emails = append(emails, store.EmailSummary{
			UID:     uint32(id),
			OpenRef: strconv.Itoa(id),
			From:    h.Get("From"),
			Subject: h.Get("Subject"),
			Date:    parseDate(h.Get("Date")),
			Seen:    false,
		})
	}

	return &store.EmailPage{
		AccountID: c.account.ID,
		Folder:    "inbox",
		Page:      page,
		Total:     total,
		Emails:    emails,
	}, nil
}

func (c *Client) GetEmail(uid int) (*store.EmailDetail, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Quit()

	count, _, err := conn.Stat()
	if err != nil {
		return nil, err
	}
	if uid < 1 || uid > count {
		return nil, fmt.Errorf("message %d not found", uid)
	}

	m, err := conn.Retr(uid)
	if err != nil {
		return nil, err
	}

	h := m.Header
	body, _ := io.ReadAll(m.Body)
	return &store.EmailDetail{
		UID:      uint32(uid),
		From:     h.Get("From"),
		To:       h.Get("To"),
		Cc:       h.Get("Cc"),
		Subject:  h.Get("Subject"),
		Date:     parseDate(h.Get("Date")),
		TextBody: string(body),
	}, nil
}

func parseDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	formats := []string{
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 MST",
	}
	for _, f := range formats {
		t, err := time.Parse(f, s)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}
