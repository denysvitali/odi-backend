package odicrypt

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	passphrase := "testpassword123"
	plaintext := "Hello, this is a secret message!"

	c, err := New(passphrase)
	if err != nil {
		t.Fatalf("Failed to create OdiCrypt: %v", err)
	}

	encrypted, err := c.Encrypt(strings.NewReader(plaintext))
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	encData, err := io.ReadAll(encrypted)
	if err != nil {
		t.Fatalf("Failed to read encrypted data: %v", err)
	}

	// Verify new format has version byte
	if encData[0] != versionV1 {
		t.Errorf("Expected version byte 0x%02x, got 0x%02x", versionV1, encData[0])
	}

	// Decrypt
	c2, _ := New(passphrase)
	decrypted, err := c2.Decrypt(io.NopCloser(bytes.NewReader(encData)))
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	decData, err := io.ReadAll(decrypted)
	if err != nil {
		t.Fatalf("Failed to read decrypted data: %v", err)
	}

	if string(decData) != plaintext {
		t.Errorf("Decryption mismatch: expected %q, got %q", plaintext, string(decData))
	}
}

func TestDecryptLegacyFormat(t *testing.T) {
	// This tests that legacy format detection works
	// Legacy format starts with a nonce (12 bytes for GCM), not 0x01
	passphrase := "testpassword"

	// Create a legacy-like ciphertext (this will fail decryption but should trigger legacy path)
	// First byte is not 0x01, so it should be detected as legacy
	legacyData := make([]byte, 50)
	legacyData[0] = 0x00 // Not version 0x01

	c, _ := New(passphrase)
	_, err := c.Decrypt(io.NopCloser(bytes.NewReader(legacyData)))

	// It should fail (because it's not valid encrypted data) but should NOT fail
	// with "data too short for salt" which would indicate it wrongly detected V1 format
	if err == nil {
		t.Error("Expected error for invalid data")
	}
	if strings.Contains(err.Error(), "data too short for salt") {
		t.Error("Legacy format was incorrectly detected as V1 format")
	}
}

func TestEncryptDecryptEmptyString(t *testing.T) {
	passphrase := "testpassword123"
	plaintext := ""

	c, err := New(passphrase)
	if err != nil {
		t.Fatalf("Failed to create OdiCrypt: %v", err)
	}

	encrypted, err := c.Encrypt(strings.NewReader(plaintext))
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	encData, err := io.ReadAll(encrypted)
	if err != nil {
		t.Fatalf("Failed to read encrypted data: %v", err)
	}

	c2, _ := New(passphrase)
	decrypted, err := c2.Decrypt(io.NopCloser(bytes.NewReader(encData)))
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	decData, err := io.ReadAll(decrypted)
	if err != nil {
		t.Fatalf("Failed to read decrypted data: %v", err)
	}

	if string(decData) != plaintext {
		t.Errorf("Decryption mismatch: expected %q, got %q", plaintext, string(decData))
	}
}

func TestEncryptDecryptLargeData(t *testing.T) {
	passphrase := "testpassword123"
	// Create 1MB of data
	plaintext := strings.Repeat("A", 1024*1024)

	c, err := New(passphrase)
	if err != nil {
		t.Fatalf("Failed to create OdiCrypt: %v", err)
	}

	encrypted, err := c.Encrypt(strings.NewReader(plaintext))
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	encData, err := io.ReadAll(encrypted)
	if err != nil {
		t.Fatalf("Failed to read encrypted data: %v", err)
	}

	c2, _ := New(passphrase)
	decrypted, err := c2.Decrypt(io.NopCloser(bytes.NewReader(encData)))
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	decData, err := io.ReadAll(decrypted)
	if err != nil {
		t.Fatalf("Failed to read decrypted data: %v", err)
	}

	if len(decData) != len(plaintext) {
		t.Errorf("Length mismatch: expected %d, got %d", len(plaintext), len(decData))
	}
}

func TestWrongPassphrase(t *testing.T) {
	plaintext := "Secret message"

	c1, _ := New("correctpassword")
	encrypted, err := c1.Encrypt(strings.NewReader(plaintext))
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	encData, _ := io.ReadAll(encrypted)

	c2, _ := New("wrongpassword")
	_, err = c2.Decrypt(io.NopCloser(bytes.NewReader(encData)))
	if err == nil {
		t.Error("Expected error when decrypting with wrong passphrase")
	}
}

func TestDecryptEmptyData(t *testing.T) {
	c, _ := New("testpassword")
	_, err := c.Decrypt(io.NopCloser(bytes.NewReader([]byte{})))
	if err == nil {
		t.Error("Expected error for empty data")
	}
}

func TestIsLegacyFormat(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "empty data returns false",
			data:     []byte{},
			expected: false,
		},
		{
			name:     "V1 format returns false",
			data:     []byte{versionV1, 0x00, 0x01, 0x02},
			expected: false,
		},
		{
			name:     "legacy format (0x00) returns true",
			data:     []byte{0x00, 0x01, 0x02, 0x03},
			expected: true,
		},
		{
			name:     "legacy format (0xFF) returns true",
			data:     []byte{0xFF, 0x01, 0x02, 0x03},
			expected: true,
		},
		{
			name:     "single byte legacy returns true",
			data:     []byte{0x00},
			expected: true,
		},
		{
			name:     "single byte V1 returns false",
			data:     []byte{versionV1},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLegacyFormat(tt.data)
			if result != tt.expected {
				t.Errorf("IsLegacyFormat(%v) = %v, expected %v", tt.data, result, tt.expected)
			}
		})
	}
}

func TestIsLegacyFormatWithEncryptedData(t *testing.T) {
	passphrase := "testpassword123"
	plaintext := "Hello, World!"

	c, _ := New(passphrase)

	// Encrypt with modern format
	encrypted, err := c.Encrypt(strings.NewReader(plaintext))
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	encData, err := io.ReadAll(encrypted)
	if err != nil {
		t.Fatalf("Failed to read encrypted data: %v", err)
	}

	// Modern encrypted data should NOT be legacy format
	if IsLegacyFormat(encData) {
		t.Error("Modern encrypted data incorrectly detected as legacy format")
	}

	// Legacy data (first byte != 0x01) should be detected as legacy
	legacyData := make([]byte, len(encData))
	copy(legacyData, encData)
	legacyData[0] = 0x00

	if !IsLegacyFormat(legacyData) {
		t.Error("Legacy data not detected as legacy format")
	}
}
