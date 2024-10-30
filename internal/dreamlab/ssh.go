package dreamlab

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

const keyDir = "keys"

// Get an existing key data or create a new key and return it.
// The returned string is an ed25519 public key
func GetCreateSSHKey(keyDir, name string) (string, error) {
	keyPath := filepath.Join(keyDir, name)
	// try existing key with the name
	pubBytes, err := os.ReadFile(keyPath + ".pub")
	if err == nil {
		return string(pubBytes), nil
	}
	// generate a new key
	ed25519Pub, ed25519Priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", nil
	}
	sshPrivateKey, err := ssh.MarshalPrivateKey(ed25519Priv, "")
	if err != nil {
		return "", err
	}
	sshPubKey, err := ssh.NewPublicKey(ed25519Pub)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(keyDir, 0750); err != nil {
		return "", nil
	}
	// write private key file
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return "", err
	}
	defer keyFile.Close()
	if err := pem.Encode(keyFile, sshPrivateKey); err != nil {
		return "", err
	}
	pubBytes = ssh.MarshalAuthorizedKey(sshPubKey)
	// write public key file
	if err := os.WriteFile(keyPath+".pub", pubBytes, 0640); err != nil {
		return "", err
	}
	return string(pubBytes), nil
}
