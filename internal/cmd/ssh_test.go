package cmd

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestEncryptPrivateKeyEncryptsUnencryptedKey(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	block, err := ssh.MarshalPrivateKey(privateKey, "test-key")
	if err != nil {
		t.Fatal(err)
	}

	encryptedKey, err := encryptPrivateKey(string(pem.EncodeToMemory(block)), "secret-passphrase", "test-key")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ssh.ParseRawPrivateKey([]byte(encryptedKey)); err == nil {
		t.Fatal("expected encrypted key to require a passphrase")
	}

	if _, err := ssh.ParseRawPrivateKeyWithPassphrase([]byte(encryptedKey), []byte("secret-passphrase")); err != nil {
		t.Fatalf("expected encrypted key to parse with passphrase: %v", err)
	}

	if _, err := ssh.ParseRawPrivateKeyWithPassphrase([]byte(encryptedKey), []byte("wrong-passphrase")); err == nil {
		t.Fatal("expected encrypted key to reject the wrong passphrase")
	}
}

func TestEncryptPrivateKeyRejectsEmptyPassword(t *testing.T) {
	if _, err := encryptPrivateKey("private key", "", "test-key"); err == nil {
		t.Fatal("expected empty password to be rejected")
	}
}
