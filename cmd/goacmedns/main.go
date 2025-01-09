package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nrdcg/goacmedns"
	"github.com/nrdcg/goacmedns/storage"
)

func main() {
	apiBase := flag.String("api", "", "ACME-DNS server API URL")
	domain := flag.String("domain", "", "Domain to register an account for")
	storagePath := flag.String("storage", "", "Path to the JSON storage file to create/update")
	allowFrom := flag.String("allowFrom", "", "List of comma separated CIDR notation networks the account is allowed to be used from")
	flag.Parse()

	if *apiBase == "" {
		log.Fatal("You must provide a non-empty -api flag")
	}

	if *domain == "" {
		log.Fatal("You must provide a non-empty -domain flag")
	}

	if *storagePath == "" {
		log.Fatal("You must provide a non-empty -storage flag")
	}

	var allowedNetworks []string
	if *allowFrom != "" {
		allowedNetworks = strings.Split(*allowFrom, ",")
	}

	err := run(*apiBase, *domain, *storagePath, allowedNetworks)
	if err != nil {
		log.Fatal(err)
	}
}

func run(apiBase, domain, storagePath string, allowedNetworks []string) error {
	client, err := goacmedns.NewClient(apiBase)
	if err != nil {
		return fmt.Errorf("could not create goacmedns client: %w", err)
	}

	st := storage.NewFile(storagePath, 0o600)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	newAcct, err := client.RegisterAccount(ctx, allowedNetworks)
	if err != nil {
		return fmt.Errorf("failed to register account: %w", err)
	}

	// Save it
	err = st.Put(ctx, domain, newAcct)
	if err != nil {
		return fmt.Errorf("failed to put account in storage: %w", err)
	}

	err = st.Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to save storage: %w", err)
	}

	log.Printf(
		"new account created for %q. "+
			"To complete setup for %q you must provision the following CNAME in your DNS zone:\n"+
			"%s CNAME %s.\n",
		domain, domain, "_acme-challenge."+domain, newAcct.FullDomain)

	return nil
}
