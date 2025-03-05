// internal/vault/vault.go
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/pbkdf2"
)

// CredentialManager handles secure storage of API keys
type CredentialManager struct {
	service  string
	fallback string // Fallback encrypted file path
}

// EncryptedCredential represents an encrypted credential
type EncryptedCredential struct {
	Ciphertext []byte `json:"ciphertext"`
	Nonce      []byte `json:"nonce"`
	Salt       []byte `json:"salt"`
}

// EncryptedStore represents the structure of the encrypted credentials file
type EncryptedStore struct {
	Credentials map[string]EncryptedCredential `json:"credentials"`
}

// NewCredentialManager creates a new credential manager
func NewCredentialManager(configDir string) (*CredentialManager, error) {
	fallbackPath := filepath.Join(configDir, "credentials.enc")
	return &CredentialManager{
		service:  "comma-git",
		fallback: fallbackPath,
	}, nil
}

// Store securely stores an API token
func (cm *CredentialManager) Store(provider, token string) error {
	// Try system keychain first
	err := keyring.Set(cm.service, provider, token)
	if err == nil {
		return nil
	}

	// Fall back to encrypted file storage with warning
	fmt.Println("Warning: Cannot use system keyring, falling back to encrypted file")
	return cm.storeFallback(provider, token)
}

// Retrieve securely retrieves an API token
func (cm *CredentialManager) Retrieve(provider string) (string, error) {
	// Try system keychain first
	token, err := keyring.Get(cm.service, provider)
	if err == nil {
		return token, nil
	}

	// Fall back to encrypted file
	return cm.retrieveFallback(provider)
}

// storeFallback stores tokens in encrypted file
func (cm *CredentialManager) storeFallback(provider, token string) error {
	// Create a better derived key using PBKDF2
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	key, err := deriveKey(salt)
	if err != nil {
		return fmt.Errorf("failed to derive key: %w", err)
	}

	// Encrypt the token
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(token), nil)

	// Create or load the store
	store := EncryptedStore{
		Credentials: make(map[string]EncryptedCredential),
	}

	// Load existing credentials if the file exists
	if _, err := os.Stat(cm.fallback); err == nil {
		data, err := ioutil.ReadFile(cm.fallback)
		if err == nil {
			if err := json.Unmarshal(data, &store); err != nil {
				// Start fresh if file is corrupted
				store.Credentials = make(map[string]EncryptedCredential)
			}
		}
	}

	// Store the new credential
	store.Credentials[provider] = EncryptedCredential{
		Ciphertext: ciphertext,
		Nonce:      nonce,
		Salt:       salt,
	}

	// Save the store
	data, err := json.Marshal(store)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(cm.fallback)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file with restricted permissions
	if err := ioutil.WriteFile(cm.fallback, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

// retrieveFallback retrieves tokens from encrypted file
func (cm *CredentialManager) retrieveFallback(provider string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(cm.fallback); os.IsNotExist(err) {
		return "", fmt.Errorf("credentials file not found")
	}

	// Read encrypted credentials
	data, err := ioutil.ReadFile(cm.fallback)
	if err != nil {
		return "", fmt.Errorf("failed to read credentials file: %w", err)
	}

	// Unmarshal the store
	var store EncryptedStore
	if err := json.Unmarshal(data, &store); err != nil {
		return "", fmt.Errorf("failed to parse credentials file: %w", err)
	}

	// Get the encrypted credential
	cred, ok := store.Credentials[provider]
	if !ok {
		return "", fmt.Errorf("no credentials found for provider: %s", provider)
	}

	// Derive the key using the stored salt
	key, err := deriveKey(cred.Salt)
	if err != nil {
		return "", fmt.Errorf("failed to derive key: %w", err)
	}

	// Decrypt the token
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, cred.Nonce, cred.Ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt value: %w", err)
	}

	return string(plaintext), nil
}

// deriveKey derives a key from machine-specific data using PBKDF2
func deriveKey(salt []byte) ([]byte, error) {
	// Get machine-specific information
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME") // For Windows
	}
	if username == "" {
		username = "unknown"
	}

	// Additional entropy
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "unknown"
	}

	// Create a machine-specific password
	machineID := fmt.Sprintf("%s:%s:%s", hostname, username, homeDir)

	// Use PBKDF2 with SHA-256 to derive a strong key
	return pbkdf2.Key([]byte(machineID), salt, 4096, 32, sha256.New), nil
}
