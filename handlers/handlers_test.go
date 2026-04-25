package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"mail0/store"
)

func setupTestRouter(t *testing.T) (*gin.Engine, *store.Store) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	s, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}

	r := gin.New()
	New(s).RegisterAPIRoutes(r.Group("/api"))
	return r, s
}

func TestListAccounts(t *testing.T) {
	r, s := setupTestRouter(t)
	s.Add(store.Account{Label: "Test", Email: "test@example.com", AccountType: store.AccountTypeIMAP, ProviderType: store.ProviderGmail})

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/accounts status = %d, want %d", w.Code, http.StatusOK)
	}
	var accounts []store.Account
	if err := json.Unmarshal(w.Body.Bytes(), &accounts); err != nil {
		t.Fatalf("unmarshal accounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("account count = %d, want 1", len(accounts))
	}
	if accounts[0].Email != "test@example.com" {
		t.Fatalf("account email = %q, want test@example.com", accounts[0].Email)
	}
	if accounts[0].Password != "" {
		t.Fatalf("password should be sanitized, got %q", accounts[0].Password)
	}
}

func TestGetAccount(t *testing.T) {
	r, s := setupTestRouter(t)
	s.Add(store.Account{Label: "Test", Email: "test@example.com", AccountType: store.AccountTypeIMAP, ProviderType: store.ProviderGmail})
	id := s.List()[0].ID

	req := httptest.NewRequest(http.MethodGet, "/api/accounts/"+id, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/accounts/:id status = %d, want %d", w.Code, http.StatusOK)
	}
	var a store.Account
	if err := json.Unmarshal(w.Body.Bytes(), &a); err != nil {
		t.Fatalf("unmarshal account: %v", err)
	}
	if a.Email != "test@example.com" {
		t.Fatalf("email = %q, want test@example.com", a.Email)
	}
}

func TestGetAccountNotFound(t *testing.T) {
	r, _ := setupTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestAddAccount(t *testing.T) {
	r, s := setupTestRouter(t)
	body := `{"label":"Test","email":"test@example.com","account_type":"imap","provider_type":"gmail","password":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /api/accounts status = %d, body = %s", w.Code, w.Body.String())
	}
	accounts := s.List()
	if len(accounts) != 1 {
		t.Fatalf("store account count = %d, want 1", len(accounts))
	}
	if accounts[0].Email != "test@example.com" || accounts[0].AccountType != store.AccountTypeIMAP {
		t.Fatalf("imported account = %+v", accounts[0])
	}
}

func TestUpdateAccount(t *testing.T) {
	r, s := setupTestRouter(t)
	s.Add(store.Account{Label: "Old", Email: "old@example.com", AccountType: store.AccountTypeIMAP, ProviderType: store.ProviderGmail})
	id := s.List()[0].ID

	body := `{"label":"New","email":"new@example.com","account_type":"imap","provider_type":"gmail"}`
	req := httptest.NewRequest(http.MethodPut, "/api/accounts/"+id, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body = %s", w.Code, w.Body.String())
	}
	a := s.List()[0]
	if a.Label != "New" || a.Email != "new@example.com" {
		t.Fatalf("updated account = %+v", a)
	}
}

func TestDeleteAccount(t *testing.T) {
	r, s := setupTestRouter(t)
	s.Add(store.Account{Label: "Test", Email: "test@example.com", AccountType: store.AccountTypeIMAP})
	id := s.List()[0].ID

	req := httptest.NewRequest(http.MethodDelete, "/api/accounts/"+id, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d", w.Code)
	}
	if len(s.List()) != 0 {
		t.Fatalf("store count = %d, want 0", len(s.List()))
	}
}

func TestImportPageGet(t *testing.T) {
	r, _ := setupTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/import", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET /api/import status = %d, want 404", w.Code)
	}
}

func TestImportEmptyDataShowsErrorAndDoesNotAddAccount(t *testing.T) {
	r, s := setupTestRouter(t)
	body := `{"type":"imap","data":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/import", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /api/import empty status = %d, want %d", w.Code, http.StatusOK)
	}
	var result store.ImportResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.Failed != 1 || len(result.Errors) == 0 || result.Errors[0] != "No import data provided" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := len(s.List()); got != 0 {
		t.Fatalf("store account count = %d, want 0", got)
	}
}

func TestImportIMAPDataAddsAccount(t *testing.T) {
	r, s := setupTestRouter(t)
	body := `{"type":"imap","data":"user@example.com----abcd efgh ijkl mnop----refresh----client----gmail"}`
	req := httptest.NewRequest(http.MethodPost, "/api/import", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /api/import valid status = %d, want %d", w.Code, http.StatusOK)
	}

	accounts := s.List()
	if len(accounts) != 1 {
		t.Fatalf("store account count = %d, want 1", len(accounts))
	}
	if accounts[0].Email != "user@example.com" || accounts[0].AccountType != store.AccountTypeIMAP || accounts[0].ProviderType != store.ProviderGmail {
		t.Fatalf("imported account = %+v, want gmail IMAP user@example.com", accounts[0])
	}
	if accounts[0].IMAPServer != "imap.gmail.com" || accounts[0].IMAPPort != 993 || !accounts[0].IMAPTLS {
		t.Fatalf("imported gmail account config = %+v, want gmail imap TLS config", accounts[0])
	}
	if accounts[0].Password != "abcdefghijklmnop" {
		t.Fatalf("imported gmail password = %q, want spaces removed", accounts[0].Password)
	}
}

func TestImportYahooDataNormalizesAppPassword(t *testing.T) {
	r, s := setupTestRouter(t)
	body := `{"type":"imap","data":"d32459208@yahoo.com----abcd efgh ijkl mnop----refresh----client----yahoo"}`
	req := httptest.NewRequest(http.MethodPost, "/api/import", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /api/import yahoo status = %d, want %d", w.Code, http.StatusOK)
	}

	accounts := s.List()
	if len(accounts) != 1 {
		t.Fatalf("store account count = %d, want 1", len(accounts))
	}
	account := accounts[0]
	if account.ProviderType != store.ProviderYahoo || account.IMAPServer != "imap.mail.yahoo.com" || account.IMAPPort != 993 || !account.IMAPTLS {
		t.Fatalf("imported yahoo account config = %+v, want yahoo imap TLS config", account)
	}
	if account.Password != "abcdefghijklmnop" {
		t.Fatalf("imported yahoo password = %q, want spaces removed", account.Password)
	}
}

func TestImportOneFailedRowShowsActionableErrorAndLogsIt(t *testing.T) {
	r, s := setupTestRouter(t)
	var logs bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() {
		log.SetOutput(previousOutput)
	})

	body := `{"type":"imap","data":"user@example.com----password----client----token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/import", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /api/import failed row status = %d, want %d", w.Code, http.StatusOK)
	}
	var result store.ImportResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	wantError := "selected IMAP but got 4 fields"
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, wantError) && strings.Contains(e, "select Microsoft Graph API") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("result errors did not include actionable error: %v", result.Errors)
	}
	if !strings.Contains(logs.String(), "Import error: line 1: "+wantError) {
		t.Fatalf("logs did not include per-line import error: %s", logs.String())
	}
	if got := len(s.List()); got != 0 {
		t.Fatalf("store account count = %d, want 0", got)
	}
}
