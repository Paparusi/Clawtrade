// vaultput stores a credential in the Clawtrade encrypted vault.
//
//	VAULTPUT_VALUE=<secret> go run ./cmd/vaultput <vault-path> <namespace> <key>
//
// Used to seed LLM provider API keys (/api_key) so the server picks them up
// without environment variables. Reads the value from VAULTPUT_VALUE so the
// secret never appears in argv.
package main

import (
	"fmt"
	"os"

	"github.com/clawtrade/clawtrade/internal/security"
)

func main() {
	args := os.Args[1:]
	if len(args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: VAULTPUT_VALUE=<secret> vaultput <vault-path> <namespace> <key>")
		os.Exit(2)
	}
	path, namespace, key := args[0], args[1], args[2]
	value := os.Getenv("VAULTPUT_VALUE")
	if value == "" {
		fmt.Fprintln(os.Stderr, "VAULTPUT_VALUE is empty")
		os.Exit(2)
	}

	var vault *security.Vault
	var err error
	if _, statErr := os.Stat(path); statErr == nil {
		vault, err = security.OpenVault(path, "clawtrade")
	} else {
		vault, err = security.NewVault(path, "clawtrade")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "vault open:", err)
		os.Exit(1)
	}
	if err := vault.Set(namespace, key, value); err != nil {
		fmt.Fprintln(os.Stderr, "vault set:", err)
		os.Exit(1)
	}
	if err := vault.Save(); err != nil {
		fmt.Fprintln(os.Stderr, "vault save:", err)
		os.Exit(1)
	}
	fmt.Printf("stored %s/%s in %s\n", namespace, key, path)
}
