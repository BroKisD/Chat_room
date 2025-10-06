package shared

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"strings"
)

func Encrypt(plain string, recipientPub *rsa.PublicKey) (string, string, error) {
	// 1) Generate random AES-256 key
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return "", "", err
	}

	// 2) AES-GCM encrypt plain -> output = nonce || ciphertext || tag (Seal does tag)
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}
	cipherText := gcm.Seal(nil, nonce, []byte(plain), nil)
	// store nonce + ciphertext
	encData := append(nonce, cipherText...)

	// 3) RSA-OAEP encrypt aesKey using recipient public key
	label := []byte("") // optional
	encKeyBytes, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, recipientPub, aesKey, label)
	if err != nil {
		return "", "", err
	}

	// 4) base64 both parts
	encKeyB64 := base64.StdEncoding.EncodeToString(encKeyBytes)
	encDataB64 := base64.StdEncoding.EncodeToString(encData)

	return encKeyB64, encDataB64, nil
}

func Decrypt(encKeyB64, encDataB64 string, priv *rsa.PrivateKey) (string, error) {
	encKeyBytes, err := base64.StdEncoding.DecodeString(encKeyB64)
	if err != nil {
		return "", err
	}
	encData, err := base64.StdEncoding.DecodeString(encDataB64)
	if err != nil {
		return "", err
	}

	// 1) RSA-OAEP decrypt aesKey
	label := []byte("")
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encKeyBytes, label)
	if err != nil {
		return "", fmt.Errorf("rsa decrypt failed: %w", err)
	}

	// 2) AES-GCM decrypt: split nonce and ciphertext (nonce size depends on GCM)
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(encData) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce := encData[:nonceSize]
	cipherText := encData[nonceSize:]
	plain, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", fmt.Errorf("aes-gcm open failed: %w", err)
	}
	return string(plain), nil
}

// GenerateRSAKeyPair generates RSA private key with bits (2048 or 4096 recommended)
func GenerateRSAKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, err
	}
	return priv, &priv.PublicKey, nil
}

// PublicKeyToPEM returns PEM-encoded PKIX public key
func PublicKeyToPEM(pub *rsa.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	}
	return pem.EncodeToMemory(block), nil
}

// ParsePublicKeyFromPEM parses PEM to *rsa.PublicKey
func ParsePublicKeyFromPEM(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("invalid public key PEM")
	}
	pubIface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := pubIface.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not RSA public key")
	}
	return pub, nil
}

func GenerateRoomKey() []byte {
	key := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		fmt.Println("Error generating room key:", err)
		return nil
	}
	return key

}

func EncryptRoomKey(pub *rsa.PublicKey, roomKey []byte) (string, error) {
	// Debug: show original roomKey (base64)
	fmt.Printf("[DEBUG] EncryptRoomKey - original roomKey (b64): %s", base64.StdEncoding.EncodeToString(roomKey))

	// RSA-OAEP encrypt directly on the raw bytes (don't convert to string)
	encKeyBytes, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, roomKey, nil)
	if err != nil {
		return "", fmt.Errorf("rsa encrypt roomKey failed: %w", err)
	}

	encKeyB64 := base64.StdEncoding.EncodeToString(encKeyBytes)
	// Debug: show base64 ciphertext being sent
	fmt.Printf("[DEBUG] EncryptRoomKey - encKeyB64 len=%d: %s", len(encKeyB64), encKeyB64)

	return encKeyB64, nil
}

func EncryptWithRoomKey(plain string, roomKey []byte) (string, string, error) {
	block, err := aes.NewCipher(roomKey)
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}

	cipherText := gcm.Seal(nonce, nonce, []byte(plain), nil)

	encDataB64 := base64.StdEncoding.EncodeToString(cipherText)

	return "", encDataB64, nil
}

func DecryptWithRoomKey(encDataB64 string, roomKey []byte) ([]byte, error) {
	encData, err := base64.StdEncoding.DecodeString(encDataB64)
	if err != nil {
		return nil, err
	}

	// 1) AES-GCM decrypt: split nonce and ciphertext (nonce size depends on GCM)
	block, err := aes.NewCipher(roomKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(encData) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce := encData[:nonceSize]
	cipherText := encData[nonceSize:]
	plain, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, fmt.Errorf("aes-gcm open failed: %w", err)
	}
	return plain, nil
}

func DecryptRoomKey(encKeyB64 string, priv *rsa.PrivateKey) []byte {
	encKeyB64 = strings.TrimSpace(encKeyB64)
	encKeyBytes, err := base64.StdEncoding.DecodeString(encKeyB64)
	if err != nil {
		fmt.Println("Failed to decode base64 encKeyB64:", err)
		return nil
	}
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encKeyBytes, nil)
	if err != nil {
		fmt.Println("Failed to rsa.DecryptOAEP room key:", err)
		return nil
	}
	fmt.Printf("[DEBUG] Decrypted room key (b64): %s", base64.StdEncoding.EncodeToString(aesKey))
	fmt.Printf("[DEBUG] Decrypted room key (hex): %x", aesKey)
	return aesKey
}
