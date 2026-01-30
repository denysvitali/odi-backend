package odicrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/pbkdf2"
)

const (
	// Version byte for new encryption format
	versionV1 byte = 0x01

	// Salt size for PBKDF2 (16 bytes = 128 bits)
	saltSize = 16

	// PBKDF2 iterations for new format (OWASP recommendation for SHA256)
	// See: https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
	pbkdf2Iterations = 600000

	// Key size for AES-256
	keySize = 32

	// Legacy settings (for backward compatibility only)
	// WARNING: These parameters are cryptographically weak and should NOT be used
	// for new encryptions. They are kept only to allow decryption of old files.
	// Issues with legacy format:
	//   - Only 4096 PBKDF2 iterations (OWASP recommends 600,000+ for SHA-1)
	//   - Uses SHA-1 for key derivation (cryptographically broken)
	//   - No salt (nil salt) - vulnerable to rainbow table attacks
	legacyIterations = 4096
)

var log = logrus.StandardLogger().WithField("package", "odicrypt")

type OdiCrypt struct {
	passphrase string
}

func New(passphrase string) (*OdiCrypt, error) {
	return &OdiCrypt{
		passphrase: passphrase,
	}, nil
}

// deriveKey derives an encryption key using PBKDF2 with the new secure parameters
func deriveKey(passphrase string, salt []byte) []byte {
	return pbkdf2.Key([]byte(passphrase), salt, pbkdf2Iterations, keySize, sha256.New)
}

// deriveLegacyKey derives an encryption key using the old insecure parameters.
// This is kept ONLY for backward compatibility when decrypting old files.
//
// SECURITY WARNING: This function uses weak cryptographic parameters:
//   - SHA-1 is cryptographically broken and should not be used
//   - Only 4096 iterations is far below OWASP recommendations (600,000+)
//   - No salt makes it vulnerable to rainbow table attacks
//
// This function MUST NOT be used for new encryptions. New code should use
// deriveKey() which uses SHA-256, 600,000 iterations, and a random salt.
func deriveLegacyKey(passphrase string) []byte {
	return pbkdf2.Key([]byte(passphrase), nil, legacyIterations, keySize, sha1.New)
}

// Encrypt encrypts data using the modern secure format (V1).
// This method always uses:
//   - PBKDF2 with SHA-256
//   - 600,000 iterations (OWASP recommendation)
//   - 16-byte random salt
//   - AES-256-GCM
//
// The output format is: [version=0x01][salt (16 bytes)][nonce][ciphertext]
// This format is NOT backward compatible with the legacy format.
//
// Note: AES-GCM does not support streaming encryption, so the entire input
// is read into memory. This may cause high memory usage for large files.
func (o *OdiCrypt) Encrypt(input io.Reader) (io.ReadSeeker, error) {
	// Generate random salt
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	// Derive key using new secure parameters
	dk := deriveKey(o.passphrase, salt)

	c, err := aes.NewCipher(dk)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	// AES-GCM requires the entire plaintext to be available at once.
	// This is a limitation of the authenticated encryption mode.
	pageBytes, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}

	cipherText := gcm.Seal(nil, nonce, pageBytes, nil)

	// New format: [version=0x01][salt (16 bytes)][nonce][ciphertext]
	var finalBytes []byte
	finalBytes = append(finalBytes, versionV1)
	finalBytes = append(finalBytes, salt...)
	finalBytes = append(finalBytes, nonce...)
	finalBytes = append(finalBytes, cipherText...)
	return bytes.NewReader(finalBytes), nil
}

// Decrypt decrypts data using either the modern V1 format or the legacy format.
// The format is auto-detected based on the version byte.
//
// Note: AES-GCM does not support streaming decryption, so the entire input
// is read into memory. This may cause high memory usage for large files.
func (o *OdiCrypt) Decrypt(objReader io.ReadCloser) (io.ReadSeeker, error) {
	// AES-GCM requires the entire ciphertext to be available at once.
	// Read all data first to determine format.
	allData, err := io.ReadAll(objReader)
	if err != nil {
		return nil, fmt.Errorf("read data: %w", err)
	}

	if len(allData) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	// Check version byte to determine format
	if allData[0] == versionV1 {
		return o.decryptV1(allData)
	}

	// Legacy format (no version byte)
	return o.decryptLegacy(allData)
}

// decryptV1 decrypts data using the new secure format
// Format: [version=0x01][salt (16 bytes)][nonce][ciphertext]
func (o *OdiCrypt) decryptV1(data []byte) (io.ReadSeeker, error) {
	// Skip version byte
	data = data[1:]

	if len(data) < saltSize {
		return nil, fmt.Errorf("data too short for salt")
	}

	// Extract salt
	salt := data[:saltSize]
	data = data[saltSize:]

	// Derive key using new secure parameters
	dk := deriveKey(o.passphrase, salt)

	c, err := aes.NewCipher(dk)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("data too short for nonce")
	}

	nonce := data[:nonceSize]
	cipherText := data[nonceSize:]

	plainText, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return bytes.NewReader(plainText), nil
}

// decryptLegacy decrypts data using the old insecure format.
// Format: [nonce][ciphertext]
//
// SECURITY WARNING: This format uses weak parameters (SHA-1, 4096 iterations, no salt).
// A deprecation warning is logged to alert the user that the file should be re-encrypted.
func (o *OdiCrypt) decryptLegacy(data []byte) (io.ReadSeeker, error) {
	log.Warn("Decrypting file using legacy format (insecure PBKDF2 parameters: SHA-1, 4096 iterations, no salt). " +
		"This file should be re-encrypted with the modern format for better security.")

	// Derive key using legacy parameters (nil salt, SHA1, 4096 iterations)
	dk := deriveLegacyKey(o.passphrase)

	c, err := aes.NewCipher(dk)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("data too short for nonce")
	}

	nonce := data[:nonceSize]
	cipherText := data[nonceSize:]

	plainText, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return bytes.NewReader(plainText), nil
}

// IsLegacyFormat returns true if the given encrypted data uses the legacy format.
// This can be used to identify files that need to be migrated to the new format.
//
// The legacy format is detected by checking if the first byte is NOT the V1 version byte.
// Note: This only checks the format indicator; it does not validate the actual data.
func IsLegacyFormat(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	return data[0] != versionV1
}
