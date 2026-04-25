package graph

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mail0/store"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	account     *store.Account
	accessToken string
	httpClient  *http.Client
}

type graphMessage struct {
	ID           string `json:"id"`
	From         struct {
		EmailAddress struct {
			Name    string `json:"name"`
			Address string `json:"address"`
		} `json:"emailAddress"`
	} `json:"from"`
	ToRecipients []struct {
		EmailAddress struct {
			Name    string `json:"name"`
			Address string `json:"address"`
		} `json:"emailAddress"`
	} `json:"toRecipients"`
	CcRecipients []struct {
		EmailAddress struct {
			Name    string `json:"name"`
			Address string `json:"address"`
		} `json:"emailAddress"`
	} `json:"ccRecipients"`
	Subject     string    `json:"subject"`
	ReceivedDateTime time.Time `json:"receivedDateTime"`
	BodyPreview string    `json:"bodyPreview"`
	Body        struct {
		ContentType string `json:"contentType"`
		Content     string `json:"content"`
	} `json:"body"`
	HasAttachments bool `json:"hasAttachments"`
	IsRead         bool `json:"isRead"`
}

type graphListResponse struct {
	Value    []graphMessage `json:"value"`
	NextLink string         `json:"@odata.nextLink"`
}

func New(a *store.Account) *Client {
	return &Client{
		account:    a,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) getToken() (string, error) {
	if c.accessToken != "" {
		return c.accessToken, nil
	}

	data := url.Values{}
	data.Set("client_id", c.account.ClientID)
	data.Set("refresh_token", c.account.RefreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequest("POST",
		"https://login.microsoftonline.com/"+c.account.TenantID+"/oauth2/v2.0/token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	c.accessToken = result.AccessToken
	return c.accessToken, nil
}

func (c *Client) do(method, path string, params url.Values) (*http.Response, error) {
	token, err := c.getToken()
	if err != nil {
		return nil, err
	}

	fullURL := "https://graph.microsoft.com/v1.0" + path
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	req, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

func (c *Client) Check() error {
	token, err := c.getToken()
	if err != nil {
		return err
	}
	if token == "" {
		return fmt.Errorf("empty access token")
	}
	return nil
}

func (c *Client) ListInbox(page, pageSize int) (*store.EmailPage, error) {
	params := url.Values{}
	params.Set("$top", fmt.Sprintf("%d", pageSize))
	params.Set("$skip", fmt.Sprintf("%d", (page-1)*pageSize))
	params.Set("$select", "id,from,toRecipients,ccRecipients,subject,receivedDateTime,bodyPreview,isRead,hasAttachments")
	params.Set("$orderby", "receivedDateTime desc")
	params.Set("$count", "true")

	resp, err := c.do("GET", "/me/mailFolders/inbox/messages", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("graph list error %d: %s", resp.StatusCode, string(body))
	}

	var result graphListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	emails := make([]store.EmailSummary, 0, len(result.Value))
	for _, m := range result.Value {
		emails = append(emails, store.EmailSummary{
			UID:     hashToUint32(m.ID),
			OpenRef: base64.RawURLEncoding.EncodeToString([]byte(m.ID)),
			From:    formatGraphAddr(m.From.EmailAddress),
			Subject: m.Subject,
			Date:    m.ReceivedDateTime,
			Seen:    m.IsRead,
		})
	}

	return &store.EmailPage{
		AccountID: c.account.ID,
		Folder:    "inbox",
		Page:      page,
		Total:     len(emails),
		Emails:    emails,
	}, nil
}

// DecodeMessageIDFromOpenRef decodes the path segment used in /mail/:openRef for Graph accounts.
func DecodeMessageIDFromOpenRef(s string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (c *Client) GetEmail(messageID string) (*store.EmailDetail, error) {
	resp, err := c.do("GET", "/me/messages/"+url.PathEscape(messageID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("graph get error %d: %s", resp.StatusCode, string(body))
	}

	var m graphMessage
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}

	detail := &store.EmailDetail{
		UID:      hashToUint32(m.ID),
		From:     formatGraphAddr(m.From.EmailAddress),
		To:       formatGraphAddrs(m.ToRecipients),
		Cc:       formatGraphAddrs(m.CcRecipients),
		Subject:  m.Subject,
		Date:     m.ReceivedDateTime,
		TextBody: m.BodyPreview,
		HTMLBody: m.BodyPreview,
	}

	return detail, nil
}

func formatGraphAddr(a struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}) string {
	if a.Name != "" {
		return a.Name + " <" + a.Address + ">"
	}
	return a.Address
}

func formatGraphAddrs(addrs []struct {
	EmailAddress struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	} `json:"emailAddress"`
}) string {
	parts := make([]string, 0, len(addrs))
	for _, a := range addrs {
		if a.EmailAddress.Name != "" {
			parts = append(parts, a.EmailAddress.Name+" <"+a.EmailAddress.Address+">")
		} else {
			parts = append(parts, a.EmailAddress.Address)
		}
	}
	return strings.Join(parts, ", ")
}

func hashToUint32(s string) uint32 {
	h := uint32(0)
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return h
}
