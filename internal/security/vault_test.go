// internal/security/vault_test.go
package security

import (
	"path/filepath"
	"testing"
)

func TestVault_StoreAndRetrieve(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.enc")

	v, err := NewVault(path, "master-password-123")
	if err != nil {
		t.Fatal(err)
	}

	err = v.Set("binance", "api_key", "my-binance-key")
	if err != nil {
		t.Fatal(err)
	}

	err = v.Set("binance", "secret", "my-binance-secret")
	if err != nil {
		t.Fatal(err)
	}

	val, err := v.Get("binance", "api_key")
	if err != nil {
		t.Fatal(err)
	}
	if val != "my-binance-key" {
		t.Errorf("expected my-binance-key, got %s", val)
	}

	err = v.Save()
	if err != nil {
		t.Fatal(err)
	}

	v2, err := OpenVault(path, "master-password-123")
	if err != nil {
		t.Fatal(err)
	}

	val2, err := v2.Get("binance", "api_key")
	if err != nil {
		t.Fatal(err)
	}
	if val2 != "my-binance-key" {
		t.Errorf("expected my-binance-key after reload, got %s", val2)
	}
}

func TestVault_WrongPassword(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.enc")

	v, _ := NewVault(path, "correct-password")
	v.Set("test", "key", "value")
	v.Save()

	_, err := OpenVault(path, "wrong-password")
	if err == nil {
		t.Error("expected error with wrong password")
	}
}

func TestVault_ListNamespaces(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.enc")

	v, _ := NewVault(path, "pass")
	v.Set("binance", "key", "val")
	v.Set("openai", "key", "val")

	ns := v.ListNamespaces()
	if len(ns) != 2 {
		t.Errorf("expected 2 namespaces, got %d", len(ns))
	}
}
