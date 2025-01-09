package storage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/nrdcg/goacmedns"
)

var testAccounts = map[string]goacmedns.Account{
	"lettuceencrypt.org": {
		FullDomain: "lettuceencrypt.org",
		SubDomain:  "tossed.lettuceencrypt.org",
		Username:   "cpu",
		Password:   "hunter2",
		ServerURL:  "https://auth.acme-dns.io",
	},
	"threeletter.agency": {
		FullDomain: "threeletter.agency",
		SubDomain:  "jobs.threeletter.agency",
		Username:   "spooky.mulder",
		Password:   "trustno1",
		ServerURL:  "https://example.org",
	},
}

func TestNewFile_nonExistentFile(t *testing.T) {
	path := filepath.Join("testdata", "non-existent.json")
	mode := os.FileMode(0o600)

	fs := NewFile(path, mode)

	if fs.path != path {
		t.Fatalf("expected fs.path = %q, got %q", path, fs.path)
	}

	if fs.mode != mode {
		t.Fatalf("expected fs.mode = %d, got %d", mode, fs.mode)
	}

	if fs.accounts == nil {
		t.Fatalf("expected accounts to be not-nil, was nil")
	}
}

func TestNewFile_withAccounts(t *testing.T) {
	fs := NewFile(filepath.Join("testdata", "accounts.json"), 0o600)

	if fs.accounts == nil {
		t.Fatalf("expected accounts to be not-nil, was nil")
	}

	if !reflect.DeepEqual(fs.accounts, testAccounts) {
		t.Errorf("expected to have accounts %#v loaded, had %#v", testAccounts, fs.accounts)
	}
}

func TestNewFile_withLegacyData(t *testing.T) {
	fs := NewFile(filepath.Join("testdata", "legacy_account.json"), 0o600)

	if fs.accounts == nil {
		t.Fatalf("expected accounts to be not-nil, was nil")
	}

	if len(fs.accounts) != 1 {
		t.Fatalf("expected a single account in the map, got %d", len(fs.accounts))
	}

	var (
		legacyAcct, testAcct goacmedns.Account
		found                bool
	)

	if legacyAcct, found = fs.accounts["threeletter.agency"]; !found {
		t.Fatalf("expected to find account but was unable to")
	}

	if legacyAcct.ServerURL != "" {
		t.Errorf("expected empty Server string from legacy account, but got %s", legacyAcct.ServerURL)
	}

	if testAcct, found = testAccounts["threeletter.agency"]; !found {
		t.Fatalf("expected to find test account for threeletter.agency, but was unable to")
	}

	// set the missing value for legacy account to be able to evaluate equivalence
	legacyAcct.ServerURL = testAcct.ServerURL

	if !reflect.DeepEqual(legacyAcct, testAcct) {
		t.Errorf("expected equivalent test and legacy accounts")
	}
}

func TestFile_Save(t *testing.T) {
	ctx := context.Background()

	file := filepath.Join(t.TempDir(), "acmedns.account")

	storage := NewFile(file, 0o600)

	for d, acct := range testAccounts {
		err := storage.Put(ctx, d, acct)
		if err != nil {
			t.Errorf("unexpected error adding account %#v to storage: %v", acct, err)
		}
	}

	err := storage.Save(ctx)
	if err != nil {
		t.Fatalf("unexpected error saving storage: %v", err)
	}

	storedJSON, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("unexpected error reading stored JSON from %q: %v", file, err)
	}

	var restoredData map[string]goacmedns.Account

	err = json.Unmarshal(storedJSON, &restoredData)
	if err != nil {
		t.Fatalf("unexpected error unmarshaling stored JSON from %q: %v", file, err)
	}

	if !reflect.DeepEqual(restoredData, testAccounts) {
		t.Errorf("Expected saved accounts and restored accounts to be equal. "+
			"Stored: %#v, Restored: %#v", testAccounts, restoredData)
	}
}

func TestFile_Fetch(t *testing.T) {
	ctx := context.Background()

	storage := NewFile("", 0)

	for d, acct := range testAccounts {
		err := storage.Put(ctx, d, acct)
		if err != nil {
			t.Errorf("unexpected error adding account %#v to storage: %v", acct, err)
		}
	}

	for d, expected := range testAccounts {
		acct, err := storage.Fetch(ctx, d)
		if err != nil {
			t.Errorf("unexpected error fetching domain %q from storage: %v", d, err)
		}

		if !reflect.DeepEqual(acct, expected) {
			t.Errorf("expected domain %q to have account %#v, had %#v\n", d, expected, acct)
		}
	}

	_, err := storage.Fetch(ctx, "doesnt-exist.example.org")
	if !errors.Is(err, ErrDomainNotFound) {
		t.Errorf("expected ErrDomainNotFound for Fetch of non-existent domain, got %v", err)
	}
}

func TestFile_FetchAll(t *testing.T) {
	ctx := context.Background()

	storage := NewFile("", 0)

	for d, acct := range testAccounts {
		err := storage.Put(ctx, d, acct)
		if err != nil {
			t.Errorf("unexpected error adding account %#v to storage: %v", acct, err)
		}
	}

	allAccounts, err := storage.FetchAll(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(allAccounts) != len(testAccounts) {
		t.Errorf("the size of fetched accounts map: %d does not match the expected: %d",
			len(allAccounts), len(testAccounts))
	}

	for d, expected := range testAccounts {
		if acct, found := allAccounts[d]; !found {
			t.Errorf("account for domain %q was not found from the fetched data", d)
		} else if !reflect.DeepEqual(acct, expected) {
			t.Errorf("expected domain %q to have account %#v, had %#v\n", d, expected, acct)
		}
	}
}
