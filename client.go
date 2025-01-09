package goacmedns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"time"
)

// defaultTimeout is used for the httpClient Timeout settings.
const defaultTimeout = 30 * time.Second

// ua is a custom user-agent identifier.
const ua = "goacmedns"

// userAgent returns a string that can be used as a HTTP request `User-Agent`
// header. It includes the `ua` string alongside the OS and architecture of the
// system.
func userAgent() string {
	return fmt.Sprintf("%s (%s; %s)", ua, runtime.GOOS, runtime.GOARCH)
}

type Register struct {
	AllowFrom []string `json:"allowfrom"`
}

type Update struct {
	SubDomain string `json:"subdomain"`
	Txt       string `json:"txt"`
}

// Storage is an interface describing the required functions for an ACME DNS
// Account storage mechanism.
type Storage interface {
	// Save will persist the `Account` data that has been `Put` so far
	Save(ctx context.Context) error
	// Put will add an `Account` for the given domain to the storage. It may not
	// be persisted until `Save` is called.
	Put(ctx context.Context, domain string, account Account) error
	// Fetch will retrieve an `Account` for the given domain from the storage. If
	// the provided domain does not have an `Account` saved in the storage
	// `ErrDomainNotFound` will be returned
	Fetch(ctx context.Context, domain string) (Account, error)
	// FetchAll retrieves all the `Account` objects from the storage and
	// returns a map that has domain names as its keys and `Account` objects
	// as values.
	FetchAll(ctx context.Context) map[string]Account
}

type Option func(c *Client)

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if c != nil {
			c.httpClient = client
		}
	}
}

type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
}

func NewClient(baseURL string, opts ...Option) (*Client, error) {
	endpoint, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse base URL: %w", err)
	}

	client := &Client{
		httpClient: &http.Client{
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       defaultTimeout,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   defaultTimeout,
					KeepAlive: defaultTimeout,
				}).DialContext,
				TLSHandshakeTimeout:   defaultTimeout,
				ResponseHeaderTimeout: defaultTimeout,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
		baseURL: endpoint,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

func (c *Client) RegisterAccount(ctx context.Context, allowFrom []string) (Account, error) {
	var register *Register
	if len(allowFrom) > 0 {
		register = &Register{AllowFrom: allowFrom}
	}

	req, err := newRequest(ctx, c.baseURL.JoinPath("register"), nil, register)
	if err != nil {
		return Account{}, err
	}

	var acct Account

	err = c.do(req, &acct)
	if err != nil {
		return Account{}, fmt.Errorf("failed to register account: %w", err)
	}

	acct.ServerURL = c.baseURL.String()

	return acct, nil
}

func (c *Client) UpdateTXTRecord(ctx context.Context, account Account, value string) error {
	update := &Update{
		SubDomain: account.SubDomain,
		Txt:       value,
	}

	headers := map[string]string{
		"X-Api-User": account.Username,
		"X-Api-Key":  account.Password,
	}

	req, err := newRequest(ctx, c.baseURL.JoinPath("update"), headers, update)
	if err != nil {
		return err
	}

	err = c.do(req, nil)
	if err != nil {
		return fmt.Errorf("failed to update TXT record: %w", err)
	}

	return nil
}

func (c *Client) do(req *http.Request, result any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do req: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(resp.Body)

		return newClientError("response error", resp.StatusCode, raw)
	}

	if result == nil {
		return nil
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}

	err = json.Unmarshal(raw, result)
	if err != nil {
		return newClientError("failed to unmarshal response", resp.StatusCode, raw)
	}

	return nil
}

func newRequest(ctx context.Context, endpoint *url.URL, headers map[string]string, payload any) (*http.Request, error) {
	buf := new(bytes.Buffer)

	if payload != nil {
		err := json.NewEncoder(buf).Encode(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to create request JSON body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), buf)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent())

	for h, v := range headers {
		req.Header.Set(h, v)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}
