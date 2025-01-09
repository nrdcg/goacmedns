package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/nrdcg/goacmedns"
)

var _ goacmedns.Storage = (*File)(nil)

// ErrDomainNotFound is returned from `Fetch` when the provided domain is not
// present in the storage.
var ErrDomainNotFound = errors.New("requested domain is not present in storage")

// File implements the `Storage` interface and persists `Accounts` to
// a JSON file on disk.
type File struct {
	// path is the filepath that the `accounts` are persisted to when the `Save`
	// function is called.
	path string
	// mode is the file mode used when the `path` JSON file must be created
	mode os.FileMode
	// accounts holds the `Account` data that has been `Put` into the storage
	accounts map[string]goacmedns.Account
}

// NewFile returns a `Storage` implementation backed by JSON content
// saved into the provided `path` on disk. The file at `path` will be created if
// required. When creating a new file the provided `mode` is used to set the
// permissions.
func NewFile(path string, mode os.FileMode) *File {
	fs := &File{
		path:     path,
		mode:     mode,
		accounts: make(map[string]goacmedns.Account),
	}

	// Opportunistically try to load the account data. Return an empty account if
	// any errors occur.
	if jsonData, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(jsonData, &fs.accounts); err != nil {
			return fs
		}
	}

	return fs
}

// Save persists the `Account` data to the File's configured path. The
// file at that path will be created with the File's mode if required.
func (f File) Save(_ context.Context) error {
	serialized, err := json.Marshal(f.accounts)
	if err != nil {
		return fmt.Errorf("fFailed to marshal account: %w", err)
	}

	if err = os.WriteFile(f.path, serialized, f.mode); err != nil {
		return fmt.Errorf("failed to write storage file: %w", err)
	}

	return nil
}

// Put saves an `Account` for the given `Domain` into the in-memory accounts of
// the File instance. The `Account` data will not be written to disk
// until the `Save` function is called.
func (f File) Put(_ context.Context, domain string, acct goacmedns.Account) error {
	f.accounts[domain] = acct

	return nil
}

// Fetch retrieves the `Account` object for the given `domain` from the
// File in-memory accounts. If the `domain` provided does not have an
// `Account` in the storage an `ErrDomainNotFound` error is returned.
func (f File) Fetch(_ context.Context, domain string) (goacmedns.Account, error) {
	if acct, exists := f.accounts[domain]; exists {
		return acct, nil
	}

	return goacmedns.Account{}, ErrDomainNotFound
}

// FetchAll retrieves all the `Account` objects from the File and
// returns a map that has domain names as its keys and `Account` objects
// as values.
func (f File) FetchAll(_ context.Context) (map[string]goacmedns.Account, error) {
	return f.accounts, nil
}
