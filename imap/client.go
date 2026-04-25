package imap

import (
	"crypto/tls"
	"fmt"
	"mail0/store"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/emersion/go-imap/v2"
	imapclient "github.com/emersion/go-imap/v2/imapclient"
)

type Client struct {
	account *store.Account
}

func New(a *store.Account) *Client {
	return &Client{account: a}
}

func (c *Client) dial() (*imapclient.Client, error) {
	a := c.account
	server := a.IMAPServer
	port := a.IMAPPort
	tlsMode := a.IMAPTLS

	if server == "" {
		s, p, t := store.ProviderConfig(a.ProviderType)
		server = s
		port = p
		tlsMode = t
	}

	addr := fmt.Sprintf("%s:%d", server, port)

	var conn net.Conn
	var err error

	if tlsMode {
		tlsConfig := &tls.Config{ServerName: server}
		conn, err = tls.Dial("tcp", addr, tlsConfig)
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	options := &imapclient.Options{}
	client := imapclient.New(conn, options)
	if err := client.Login(a.Email, loginPassword(a)).Wait(); err != nil {
		client.Close()
		return nil, formatLoginError(a, err)
	}
	return client, nil
}

func loginPassword(a *store.Account) string {
	if isAppPasswordProvider(a) {
		return strings.Join(strings.Fields(a.Password), "")
	}
	return a.Password
}

func formatLoginError(a *store.Account, err error) error {
	if isYahooIMAP(a) {
		return fmt.Errorf("login: %w; Yahoo IMAP requires an app password, not the normal account password. Generate a Yahoo app password and use it without spaces", err)
	}
	if isGmailIMAP(a) {
		return fmt.Errorf("login: %w; Gmail IMAP usually requires IMAP to be enabled and a Google app password, not the normal Google account password. If 2-Step Verification is enabled, generate an app password and use it without spaces; also verify the email address is correct", err)
	}
	return fmt.Errorf("login: %w", err)
}

func isAppPasswordProvider(a *store.Account) bool {
	return isYahooIMAP(a) || isGmailIMAP(a)
}

func isYahooIMAP(a *store.Account) bool {
	if a.ProviderType == store.ProviderYahoo {
		return true
	}
	return strings.Contains(strings.ToLower(a.IMAPServer), "yahoo")
}

func isGmailIMAP(a *store.Account) bool {
	if a.ProviderType == store.ProviderGmail {
		return true
	}
	server := strings.ToLower(a.IMAPServer)
	return strings.Contains(server, "gmail") || strings.Contains(server, "googlemail")
}

func (c *Client) Check() error {
	client, err := c.dial()
	if err != nil {
		return err
	}
	defer client.Logout()
	defer client.Close()
	return nil
}

func (c *Client) ListInbox(page, pageSize int) (*store.EmailPage, error) {
	client, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer client.Logout()
	defer client.Close()

	return c.listFolder(client, "INBOX", page, pageSize)
}

func (c *Client) ListFolder(folder string, page, pageSize int) (*store.EmailPage, error) {
	client, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer client.Logout()
	defer client.Close()

	return c.listFolder(client, folder, page, pageSize)
}

func (c *Client) listFolder(client *imapclient.Client, folder string, page, pageSize int) (*store.EmailPage, error) {
	mbox, err := client.Select(folder, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("select %s: %w", folder, err)
	}

	total := int(mbox.NumMessages)
	if total == 0 {
		return &store.EmailPage{
			AccountID: c.account.ID,
			Folder:    folder,
			Page:      page,
			Total:     0,
			Emails:    []store.EmailSummary{},
		}, nil
	}

	start := total - (page * pageSize)
	if start < 1 {
		start = 1
	}
	end := start + pageSize - 1
	if end > total {
		end = total
	}

	if start > end {
		return &store.EmailPage{
			AccountID: c.account.ID, Folder: folder, Page: page, Total: total,
			Emails: []store.EmailSummary{},
		}, nil
	}

	seqSet := imap.SeqSet{}
	seqSet.AddRange(uint32(start), uint32(end))

	fetchOptions := &imap.FetchOptions{
		Envelope:   true,
		Flags:      true,
		RFC822Size: true,
	}
	messages, err := client.Fetch(seqSet, fetchOptions).Collect()
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	emails := make([]store.EmailSummary, 0, len(messages))
	for _, msg := range messages {
		seqNum := msg.SeqNum
		seen := false
		for _, f := range msg.Flags {
			if f == imap.FlagSeen {
				seen = true
				break
			}
		}
		summary := store.EmailSummary{
			UID:     seqNum,
			OpenRef: strconv.FormatUint(uint64(seqNum), 10),
			From:    formatAddress(msg.Envelope.From),
			Subject: msg.Envelope.Subject,
			Date:    msg.Envelope.Date,
			Seen:    seen,
			Size:    uint32(msg.RFC822Size),
		}
		emails = append(emails, summary)
	}

	sort.Slice(emails, func(i, j int) bool {
		return emails[i].UID > emails[j].UID
	})

	return &store.EmailPage{
		AccountID: c.account.ID, Folder: folder, Page: page, Total: total,
		Emails: emails,
	}, nil
}

func (c *Client) GetEmail(folder string, uid uint32) (*store.EmailDetail, error) {
	client, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer client.Logout()
	defer client.Close()

	_, err = client.Select(folder, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}

	seqSet := imap.SeqSet{}
	seqSet.AddNum(uid)

	sectionText := []string{"BODY[TEXT]", "BODY[1]", "BODY[2]"}
	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		BodySection: []*imap.FetchItemBodySection{
			{Peek: true, Specifier: imap.PartSpecifierText},
		},
	}
	_ = sectionText

	messages, err := client.Fetch(seqSet, fetchOptions).Collect()
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("message %d not found", uid)
	}

	msg := messages[0]
	detail := &store.EmailDetail{
		UID:     uid,
		From:    formatAddress(msg.Envelope.From),
		To:      formatAddress(msg.Envelope.To),
		Cc:      formatAddress(msg.Envelope.Cc),
		Subject: msg.Envelope.Subject,
		Date:    msg.Envelope.Date,
	}

	for _, section := range msg.BodySection {
		if len(section.Bytes) > 0 {
			detail.HTMLBody = string(section.Bytes)
		}
	}

	return detail, nil
}

func formatAddress(addrs []imap.Address) string {
	result := ""
	for i, a := range addrs {
		if i > 0 {
			result += ", "
		}
		if a.Name != "" {
			result += a.Name + " <" + a.Addr() + ">"
		} else {
			result += a.Addr()
		}
	}
	return result
}
