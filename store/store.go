package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type AccountType string

const (
	AccountTypeIMAP  AccountType = "imap"
	AccountTypeGraph AccountType = "graph"
	AccountTypePOP3  AccountType = "pop3"
)

type ProviderType string

const (
	ProviderGmail   ProviderType = "gmail"
	ProviderYahoo   ProviderType = "yahoo"
	ProviderQQ      ProviderType = "qq"
	Provider163     ProviderType = "163"
	ProviderOutlook ProviderType = "outlook"
	ProviderCustom  ProviderType = "custom"
)

type Account struct {
	ID           string      `json:"id"`
	Label        string      `json:"label"`
	Email        string      `json:"email"`
	AccountType  AccountType `json:"account_type"`
	ProviderType ProviderType `json:"provider_type"`

	// IMAP settings (when type=imap)
	IMAPServer   string `json:"imap_server,omitempty"`
	IMAPPort     int    `json:"imap_port,omitempty"`
	IMAPTLS      bool   `json:"imap_tls,omitempty"`
	Password     string `json:"password,omitempty"`

	// Graph API settings (when type=graph)
	ClientID     string `json:"client_id,omitempty"`
	TenantID     string `json:"tenant_id,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`

	// POP3 settings (when type=pop3)
	POP3Server string `json:"pop3_server,omitempty"`
	POP3Port   int    `json:"pop3_port,omitempty"`
	POP3TLS    bool   `json:"pop3_tls,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type EmailSummary struct {
	UID       uint32    `json:"uid"`
	OpenRef   string    `json:"open_ref,omitempty"` // path segment for /mail/:openRef (Graph: base64url of message id; IMAP/POP3: decimal uid)
	From      string    `json:"from"`
	Subject   string    `json:"subject"`
	Date      time.Time `json:"date"`
	Seen      bool      `json:"seen"`
	Size      uint32    `json:"size"`
}

type EmailPage struct {
	AccountID string         `json:"account_id"`
	Folder    string         `json:"folder"`
	Page      int            `json:"page"`
	Total     int            `json:"total"`
	Emails    []EmailSummary `json:"emails"`
}

type EmailDetail struct {
	UID        uint32              `json:"uid"`
	From       string              `json:"from"`
	To         string              `json:"to"`
	Cc         string              `json:"cc"`
	Subject    string              `json:"subject"`
	Date       time.Time           `json:"date"`
	TextBody   string              `json:"text_body"`
	HTMLBody   string              `json:"html_body"`
	Attachments []AttachmentInfo   `json:"attachments"`
}

type AttachmentInfo struct {
	Filename string `json:"filename"`
	Size     uint32 `json:"size"`
	MimeType string `json:"mime_type"`
}

type ImportRequest struct {
	Data string `json:"data"` // newline-separated entries
}

type ImportResult struct {
	Total    int      `json:"total"`
	Success  int      `json:"success"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors"`
}

type Store struct {
	mu       sync.RWMutex
	path     string
	Accounts []Account `json:"accounts"`
}

func New(path string) (*Store, error) {
	s := &Store{path: path, Accounts: []Account{}}
	if err := s.load(); err != nil {
		// If file doesn't exist, start fresh
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.Accounts)
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.Accounts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *Store) List() []Account {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Account, len(s.Accounts))
	copy(result, s.Accounts)
	return result
}

func (s *Store) Get(id string) (Account, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.Accounts {
		if a.ID == id {
			return a, true
		}
	}
	return Account{}, false
}

func (s *Store) Add(a Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
	s.Accounts = append(s.Accounts, a)
	return s.save()
}

func (s *Store) Update(id string, a Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.Accounts {
		if existing.ID == id {
			a.ID = id
			a.CreatedAt = existing.CreatedAt
			a.UpdatedAt = time.Now()
			s.Accounts[i] = a
			return s.save()
		}
	}
	return fmt.Errorf("account %s not found", id)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, a := range s.Accounts {
		if a.ID == id {
			s.Accounts = append(s.Accounts[:i], s.Accounts[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("account %s not found", id)
}

func ProviderConfig(p ProviderType) (server string, port int, tls bool) {
	switch p {
	case ProviderGmail:
		return "imap.gmail.com", 993, true
	case ProviderYahoo:
		return "imap.mail.yahoo.com", 993, true
	case ProviderQQ:
		return "imap.qq.com", 993, true
	case Provider163:
		return "imap.163.com", 993, true
	case ProviderOutlook:
		return "outlook.office365.com", 993, true
	default:
		return "", 993, true
	}
}

func POP3ProviderConfig(p ProviderType) (string, int, bool) {
	switch p {
	case ProviderGmail:
		return "pop.gmail.com", 995, true
	case ProviderYahoo:
		return "pop.mail.yahoo.com", 995, true
	case ProviderQQ:
		return "pop.qq.com", 995, true
	case Provider163:
		return "pop.163.com", 995, true
	case ProviderOutlook:
		return "outlook.office365.com", 995, true
	default:
		return "", 995, true
	}
}
