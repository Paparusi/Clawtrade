// internal/security/vault.go
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"

	"golang.org/x/crypto/pbkdf2"
)

const (
	pbkdf2Iterations = 100000
	saltSize         = 32
	keySize          = 32
)

type Vault struct {
	mu   sync.RWMutex
	path string
	key  []byte
	salt []byte
	data map[string]map[string]string
}

type vaultFile struct {
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

func deriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, keySize, sha256.New)
}

func NewVault(path, password string) (*Vault, error) {
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	return &Vault{
		path: path,
		key:  deriveKey(password, salt),
		salt: salt,
		data: make(map[string]map[string]string),
	}, nil
}

func OpenVault(path, password string) (*Vault, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read vault file: %w", err)
	}

	var vf vaultFile
	if err := json.Unmarshal(raw, &vf); err != nil {
		return nil, fmt.Errorf("parse vault file: %w", err)
	}

	key := deriveKey(password, vf.Salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	plaintext, err := aesGCM.Open(nil, vf.Nonce, vf.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt vault (wrong password?): %w", err)
	}

	var data map[string]map[string]string
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, fmt.Errorf("parse vault data: %w", err)
	}

	return &Vault{path: path, key: key, salt: vf.Salt, data: data}, nil
}

func (v *Vault) Set(namespace, key, value string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.data[namespace] == nil {
		v.data[namespace] = make(map[string]string)
	}
	v.data[namespace][key] = value
	return nil
}

func (v *Vault) Get(namespace, key string) (string, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	ns, ok := v.data[namespace]
	if !ok {
		return "", fmt.Errorf("namespace %q not found", namespace)
	}
	val, ok := ns[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in namespace %q", key, namespace)
	}
	return val, nil
}

func (v *Vault) Delete(namespace, key string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if ns, ok := v.data[namespace]; ok {
		delete(ns, key)
		if len(ns) == 0 {
			delete(v.data, namespace)
		}
	}
}

func (v *Vault) ListNamespaces() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	ns := make([]string, 0, len(v.data))
	for k := range v.data {
		ns = append(ns, k)
	}
	sort.Strings(ns)
	return ns
}

func (v *Vault) Save() error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	plaintext, err := json.Marshal(v.data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	block, err := aes.NewCipher(v.key)
	if err != nil {
		return fmt.Errorf("create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nil, nonce, plaintext, nil)

	vf := vaultFile{Salt: v.salt, Nonce: nonce, Ciphertext: ciphertext}
	raw, err := json.Marshal(vf)
	if err != nil {
		return fmt.Errorf("marshal vault file: %w", err)
	}

	return os.WriteFile(v.path, raw, 0600)
}
