# goacmedns

A Go library to handle [acme-dns](https://github.com/joohoi/acme-dns) client communication and persistent account storage.

[![CI Status](https://github.com/nrdcg/goacmedns/workflows/Go/badge.svg)](https://github.com/nrdcg/goacmedns/actions?query=workflow%3AGo)
[![Lint Status](https://github.com/nrdcg/goacmedns/workflows/golangci-lint/badge.svg)](https://github.com/nrdcg/goacmedns/actions?query=workflow%3Agolangci-lint)
[![Go Report Card](https://goreportcard.com/badge/github.com/nrdcg/goacmedns)](https://goreportcard.com/report/github.com/nrdcg/goacmedns)

You may also be interested in a Python equivalent [pyacmedns](https://github.com/joohoi/pyacmedns/).

## Installation

Once you have [installed Go](https://golang.org/doc/install) 1.21+ you can install `goacmedns` with `go install`:

```bash
go install github.com/nrdcg/goacmedns/cmd/goacmedns@latest
```

## Usage

The following is a short example of using the library to update a TXT record served by an `acme-dns` instance.

```go
package main

import (
	"context"
	"errors"
	"log"

	"github.com/nrdcg/goacmedns"
	"github.com/nrdcg/goacmedns/storage"
)

const (
	domain = "your.example.org"
)

var (
	whitelistedNetworks = []string{"192.168.11.0/24", "[::1]/128"}
)

func main() {
	// Initialize the client. Point it towards your acme-dns instance.
	client, err := goacmedns.NewClient("https://auth.acme-dns.io")

	ctx := context.Background()

	// Initialize the storage.
	// If the file does not exist, it will be automatically created.
	st := storage.NewFile("/tmp/storage.json", 0600)

	// Check if credentials were previously saved for your domain.
	account, err := st.Fetch(ctx, domain)
	if err != nil {
		if !errors.Is(err, storage.ErrDomainNotFound) {
			log.Fatal(err)
		}

		// The account did not exist.
		// Let's create a new one The whitelisted networks parameter is optional and can be nil.
		newAcct, err := client.RegisterAccount(ctx, whitelistedNetworks)
		if err != nil {
			log.Fatal(err)
		}

		// Save it
		err = st.Put(ctx, domain, newAcct)
		if err != nil {
			log.Fatalf("Failed to put account in storage: %v", err)
		}

		err = st.Save(ctx)
		if err != nil {
			log.Fatalf("Failed to save storage: %v", err)
		}

		account = newAcct
	}

	// Update the acme-dns TXT record.
	err = client.UpdateTXTRecord(ctx, account, "___validation_token_recieved_from_the_ca___")
	if err != nil {
		log.Fatal(err)
	}
}
```

## Pre-Registration

When using `goacmedns` with an ACME client hook
it may be desirable to do the initial ACME-DNS account creation and CNAME delegation ahead of time.

The `goacmedns` command line utility provides an easy way to do this:

```bash
go install github.com/nrdcg/goacmedns/cmd/goacmedns@latest

goacmedns -api http://10.0.0.1:4443 -domain example.com -allowFrom 192.168.100.1/24,1.2.3.4/32,2002:c0a8:2a00::0/40 -storage /tmp/example.storage.json
```

This will register an account for `example.com` that is only usable from the specified CIDR `-allowFrom` networks with the ACME-DNS server at `http://10.0.0.1:4443`,
saving the account details in `/tmp/example.storage.json` and printing the required CNAME record for the `example.com` DNS zone to stdout.
