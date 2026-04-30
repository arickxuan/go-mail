package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"mail0/graph"
	"mail0/imap"
	"mail0/pop3"
	"mail0/store"
)

type Handler struct {
	store *store.Store
}

func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) RegisterAPIRoutes(r *gin.RouterGroup) {
	r.GET("/accounts", h.listAccounts)
	r.GET("/accounts/:id", h.getAccount)
	r.POST("/accounts", h.addAccount)
	r.PUT("/accounts/:id", h.updateAccount)
	r.DELETE("/accounts/:id", h.deleteAccount)
	r.POST("/accounts/:id/check", h.checkAccount)
	r.GET("/accounts/:id/inbox", h.listInbox)
	r.GET("/accounts/:id/mail/:uid", h.viewEmail)
	r.POST("/import", h.importAccounts)
}

func sanitizeAccount(a store.Account) store.Account {
	a.Password = ""
	a.RefreshToken = ""
	return a
}

func (h *Handler) listAccounts(c *gin.Context) {
	accounts := h.store.List()
	result := make([]store.Account, len(accounts))
	for i, a := range accounts {
		result[i] = sanitizeAccount(a)
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) getAccount(c *gin.Context) {
	id := c.Param("id")
	a, ok := h.store.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}
	c.JSON(http.StatusOK, sanitizeAccount(a))
}

type createAccountRequest struct {
	Label        string             `json:"label"`
	Email        string             `json:"email"`
	AccountType  store.AccountType  `json:"account_type"`
	ProviderType store.ProviderType `json:"provider_type"`
	Password     string             `json:"password,omitempty"`
	IMAPServer   string             `json:"imap_server,omitempty"`
	IMAPPort     int                `json:"imap_port,omitempty"`
	IMAPTLS      bool               `json:"imap_tls,omitempty"`
	POP3Server   string             `json:"pop3_server,omitempty"`
	POP3Port     int                `json:"pop3_port,omitempty"`
	POP3TLS      bool               `json:"pop3_tls,omitempty"`
	ClientID     string             `json:"client_id,omitempty"`
	TenantID     string             `json:"tenant_id,omitempty"`
	RefreshToken string             `json:"refresh_token,omitempty"`
}

func (h *Handler) addAccount(c *gin.Context) {
	var req createAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	a := store.Account{
		Label:        req.Label,
		Email:        req.Email,
		AccountType:  req.AccountType,
		ProviderType: req.ProviderType,
	}

	switch req.AccountType {
	case store.AccountTypeIMAP:
		a.Password = normalizeProviderPassword(req.ProviderType, req.Password)
		if req.IMAPServer != "" {
			a.IMAPServer = req.IMAPServer
			a.IMAPPort = req.IMAPPort
		} else {
			s, p, tls := store.ProviderConfig(req.ProviderType)
			a.IMAPServer = s
			a.IMAPPort = p
			a.IMAPTLS = tls
		}
		if req.IMAPServer != "" {
			a.IMAPTLS = req.IMAPTLS
		}
	case store.AccountTypeGraph:
		a.ClientID = req.ClientID
		a.TenantID = req.TenantID
		a.RefreshToken = req.RefreshToken
	case store.AccountTypePOP3:
		a.Password = normalizeProviderPassword(req.ProviderType, req.Password)
		if req.POP3Server != "" {
			a.POP3Server = req.POP3Server
			a.POP3Port = req.POP3Port
		} else {
			s, p, tls := store.POP3ProviderConfig(req.ProviderType)
			a.POP3Server = s
			a.POP3Port = p
			a.POP3TLS = tls
		}
		if req.POP3Server != "" {
			a.POP3TLS = req.POP3TLS
		}
	}

	if err := h.store.Add(a); err != nil {
		log.Printf("Error adding account: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add account"})
		return
	}

	log.Printf("Account added: %s (%s)", a.Email, a.AccountType)
	c.JSON(http.StatusOK, sanitizeAccount(a))
}

func (h *Handler) updateAccount(c *gin.Context) {
	id := c.Param("id")
	a, ok := h.store.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	var req createAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	a.Label = req.Label
	a.Email = req.Email

	switch a.AccountType {
	case store.AccountTypeIMAP:
		if req.Password != "" {
			a.Password = normalizeProviderPassword(a.ProviderType, req.Password)
		}
		if req.IMAPServer != "" {
			a.IMAPServer = req.IMAPServer
			a.IMAPPort = req.IMAPPort
			a.IMAPTLS = req.IMAPTLS
		}
	case store.AccountTypeGraph:
		a.ClientID = req.ClientID
		a.TenantID = req.TenantID
		if req.RefreshToken != "" {
			a.RefreshToken = req.RefreshToken
		}
	case store.AccountTypePOP3:
		if req.Password != "" {
			a.Password = normalizeProviderPassword(a.ProviderType, req.Password)
		}
		if req.POP3Server != "" {
			a.POP3Server = req.POP3Server
			a.POP3Port = req.POP3Port
			a.POP3TLS = req.POP3TLS
		}
	}

	if err := h.store.Update(id, a); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sanitizeAccount(a))
}

func (h *Handler) deleteAccount(c *gin.Context) {
	id := c.Param("id")
	if err := h.store.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *Handler) checkAccount(c *gin.Context) {
	id := c.Param("id")
	a, ok := h.store.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	var err error
	switch a.AccountType {
	case store.AccountTypeIMAP:
		err = imap.New(&a).Check()
	case store.AccountTypeGraph:
		err = graph.New(&a).Check()
	case store.AccountTypePOP3:
		err = pop3.New(&a).Check()
	}

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "error", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type inboxResponse struct {
	Account store.Account        `json:"account"`
	Page    int                  `json:"page"`
	Total   int                  `json:"total"`
	Emails  []store.EmailSummary `json:"emails"`
}

func (h *Handler) listInbox(c *gin.Context) {
	id := c.Param("id")
	a, ok := h.store.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		page, _ = strconv.Atoi(pageStr)
	}

	var result *store.EmailPage
	var err error

	switch a.AccountType {
	case store.AccountTypeIMAP:
		result, err = imap.New(&a).ListInbox(page, 50)
	case store.AccountTypeGraph:
		result, err = graph.New(&a).ListInbox(page, 50)
	case store.AccountTypePOP3:
		result, err = pop3.New(&a).ListInbox(page, 50)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported account type"})
		return
	}

	if err != nil {
		log.Printf("Inbox error for %s: %v", a.Email, err)
		c.JSON(http.StatusOK, inboxResponse{
			Account: sanitizeAccount(a),
			Page:    page,
			Total:   0,
			Emails:  []store.EmailSummary{},
		})
		return
	}

	c.JSON(http.StatusOK, inboxResponse{
		Account: sanitizeAccount(a),
		Page:    result.Page,
		Total:   result.Total,
		Emails:  result.Emails,
	})
}

func (h *Handler) viewEmail(c *gin.Context) {
	id := c.Param("id")
	uidStr := c.Param("uid")

	a, ok := h.store.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	var detail *store.EmailDetail
	var err error
	switch a.AccountType {
	case store.AccountTypeIMAP:
		uid, perr := strconv.ParseUint(uidStr, 10, 32)
		if perr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UID"})
			return
		}
		detail, err = imap.New(&a).GetEmail("INBOX", uint32(uid))
	case store.AccountTypeGraph:
		messageID, derr := graph.DecodeMessageIDFromOpenRef(uidStr)
		if derr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message reference"})
			return
		}
		detail, err = graph.New(&a).GetEmail(messageID)
	case store.AccountTypePOP3:
		uid, perr := strconv.ParseUint(uidStr, 10, 32)
		if perr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UID"})
			return
		}
		detail, err = pop3.New(&a).GetEmail(int(uid))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported account type"})
		return
	}

	if err != nil {
		log.Printf("View email error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch email: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, detail)
}

type importRequest struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func (h *Handler) importAccounts(c *gin.Context) {
	var req importRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := store.ImportResult{Errors: []string{}}
	data := strings.TrimSpace(req.Data)
	if data == "" {
		result.Failed = 1
		result.Errors = append(result.Errors, "No import data provided")
		c.JSON(http.StatusOK, result)
		return
	}

	lines := strings.Split(data, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result.Total++

		a, err := parseImportLine(req.Type, line)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: %v", i+1, err))
			continue
		}

		if err := h.store.Add(a); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: save error: %v", i+1, err))
			continue
		}
		result.Success++
	}

	log.Printf("Import complete: %d total, %d success, %d failed", result.Total, result.Success, result.Failed)
	for _, errMsg := range result.Errors {
		log.Printf("Import error: %s", errMsg)
	}
	c.JSON(http.StatusOK, result)
}

func parseImportLine(accType, line string) (store.Account, error) {
	fields := strings.Split(line, "----")

	switch accType {
	case "graph":
		if len(fields) != 4 {
			return store.Account{}, fmt.Errorf("selected Graph API but got %d fields; expected email----password----client_id----token", len(fields))
		}
		email := strings.TrimSpace(fields[0])
		clientID := strings.TrimSpace(fields[2])
		refreshToken := strings.TrimSpace(fields[3])
		if email == "" || clientID == "" || refreshToken == "" {
			return store.Account{}, fmt.Errorf("selected Graph API but email, client_id, and token are required")
		}
		return store.Account{
			AccountType:  store.AccountTypeGraph,
			Email:        email,
			Password:     strings.TrimSpace(fields[1]),
			ClientID:     clientID,
			RefreshToken: refreshToken,
			TenantID:     "common",
			Label:        email,
		}, nil
	case "imap":
		if len(fields) != 5 {
			return store.Account{}, fmt.Errorf("selected IMAP but got %d fields; expected email----password----refresh_token----client_id----provider (provider: outlook/yahoo/gmail/qq/163). If this is a 4-field Graph row, select Microsoft Graph API", len(fields))
		}
		email := strings.TrimSpace(fields[0])
		provider := store.ProviderType(strings.ToLower(strings.TrimSpace(fields[4])))
		password := normalizeProviderPassword(provider, fields[1])
		if email == "" || password == "" {
			return store.Account{}, fmt.Errorf("selected IMAP but email and password are required")
		}
		if !isKnownIMAPProvider(provider) {
			return store.Account{}, fmt.Errorf("unsupported IMAP provider %q; expected one of outlook/yahoo/gmail/qq/163", provider)
		}
		imapServer, imapPort, imapTLS := store.ProviderConfig(provider)

		return store.Account{
			AccountType:  store.AccountTypeIMAP,
			ProviderType: provider,
			Email:        email,
			Password:     password,
			RefreshToken: strings.TrimSpace(fields[2]),
			ClientID:     strings.TrimSpace(fields[3]),
			IMAPServer:   imapServer,
			IMAPPort:     imapPort,
			IMAPTLS:      imapTLS,
			Label:        email,
		}, nil
	default:
		return store.Account{}, fmt.Errorf("unsupported import type %q; expected imap or graph", accType)
	}
}

func normalizeProviderPassword(provider store.ProviderType, password string) string {
	password = strings.TrimSpace(password)
	if provider == store.ProviderYahoo || provider == store.ProviderGmail {
		return strings.Join(strings.Fields(password), "")
	}
	return password
}

func isKnownIMAPProvider(provider store.ProviderType) bool {
	switch provider {
	case store.ProviderOutlook, store.ProviderYahoo, store.ProviderGmail, store.ProviderQQ, store.Provider163:
		return true
	default:
		return false
	}
}

func (h *Handler) GetMailHtml(c *gin.Context) {
	html, err := os.ReadFile("web/mail.html")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read file: " + err.Error()})
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", html)

}
