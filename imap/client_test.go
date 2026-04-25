package imap

import (
	"errors"
	"strings"
	"testing"

	"mail0/store"
)

func TestFormatLoginErrorAddsYahooAppPasswordHint(t *testing.T) {
	account := &store.Account{
		Email:        "d32459208@yahoo.com",
		ProviderType: store.ProviderYahoo,
		IMAPServer:   "imap.mail.yahoo.com",
	}

	err := formatLoginError(account, errors.New("imap: NO [AUTHENTICATIONFAILED] LOGIN Invalid credentials"))

	msg := err.Error()
	if !strings.Contains(msg, "AUTHENTICATIONFAILED") {
		t.Fatalf("login error = %q, want original authentication failure", msg)
	}
	if !strings.Contains(msg, "Yahoo IMAP requires an app password") {
		t.Fatalf("login error = %q, want Yahoo app password hint", msg)
	}
}

func TestLoginPasswordNormalizesYahooAppPasswordSpaces(t *testing.T) {
	account := &store.Account{
		ProviderType: store.ProviderYahoo,
		Password:     "abcd efgh ijkl mnop",
	}

	if got := loginPassword(account); got != "abcdefghijklmnop" {
		t.Fatalf("loginPassword() = %q, want spaces removed", got)
	}
}

func TestFormatLoginErrorAddsGmailAppPasswordHint(t *testing.T) {
	account := &store.Account{
		Email:        "user@gmail.com",
		ProviderType: store.ProviderGmail,
		IMAPServer:   "imap.gmail.com",
	}

	err := formatLoginError(account, errors.New("imap: NO [AUTHENTICATIONFAILED] Invalid credentials (Failure)"))

	msg := err.Error()
	if !strings.Contains(msg, "AUTHENTICATIONFAILED") {
		t.Fatalf("login error = %q, want original authentication failure", msg)
	}
	if !strings.Contains(msg, "Gmail IMAP usually requires IMAP to be enabled") {
		t.Fatalf("login error = %q, want Gmail IMAP hint", msg)
	}
	if !strings.Contains(msg, "Google app password") {
		t.Fatalf("login error = %q, want Google app password hint", msg)
	}
}

func TestLoginPasswordNormalizesGmailAppPasswordSpaces(t *testing.T) {
	account := &store.Account{
		ProviderType: store.ProviderGmail,
		Password:     "abcd efgh ijkl mnop",
	}

	if got := loginPassword(account); got != "abcdefghijklmnop" {
		t.Fatalf("loginPassword() = %q, want spaces removed", got)
	}
}

func TestFormatLoginErrorForOtherProviderKeepsGenericLoginError(t *testing.T) {
	account := &store.Account{
		Email:        "user@example.com",
		ProviderType: store.ProviderOutlook,
		IMAPServer:   "outlook.office365.com",
	}

	err := formatLoginError(account, errors.New("invalid credentials"))

	msg := err.Error()
	if !strings.Contains(msg, "login: invalid credentials") {
		t.Fatalf("login error = %q, want original login error", msg)
	}
	if strings.Contains(msg, "Yahoo IMAP requires an app password") {
		t.Fatalf("login error = %q, did not want Yahoo hint", msg)
	}
	if strings.Contains(msg, "Gmail IMAP usually requires") {
		t.Fatalf("login error = %q, did not want Gmail hint", msg)
	}
}
