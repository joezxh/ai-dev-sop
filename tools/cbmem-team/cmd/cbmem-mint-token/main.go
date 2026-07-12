// Package main provides a tiny CLI to mint a JWT for an existing user.
//
// Usage:
//
//	cbmem-mint-token --secret $JWT_SECRET --user alice --ttl 720h
//
// Useful when you do not want to expose /admin/* over the network.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"cbmem-team/internal/auth"
)

func main() {
	secret := flag.String("secret", "", "JWT shared secret")
	user := flag.String("user", "", "user id (sub claim)")
	ttl := flag.Duration("ttl", 720*time.Hour, "token lifetime")
	flag.Parse()
	if *secret == "" || *user == "" {
		fmt.Fprintln(os.Stderr, "usage: cbmem-mint-token --secret $JWT_SECRET --user alice [--ttl 720h]")
		os.Exit(2)
	}
	v := auth.NewVerifier([]byte(*secret))
	tok, err := v.Sign(*user, *ttl)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(tok)
}
