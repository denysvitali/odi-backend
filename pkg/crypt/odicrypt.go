package odicrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

type OdiCrypt struct {
	gcm           cipher.AEAD
	encryptionKey []byte
}

func New(passphrase string) (*OdiCrypt, error) {
	dk := pbkdf2.Key([]byte(passphrase), nil, 4096, 32, sha1.New)

	encryptionKey := dk[:32]
	c, err := aes.NewCipher(dk[:32])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	o := &OdiCrypt{}
	o.gcm = gcm
	o.encryptionKey = encryptionKey
	return o, nil
}
func (o *OdiCrypt) Encrypt(input io.Reader) (io.ReadSeeker, error) {
	nonce := make([]byte, o.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	pageBytes, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	cipherText := o.gcm.Seal(nil, nonce, pageBytes, nil)

	var finalBytes []byte
	finalBytes = append(finalBytes, nonce...)
	finalBytes = append(finalBytes, cipherText...)
	return bytes.NewReader(finalBytes), nil
}

func (o *OdiCrypt) Decrypt(objReader io.ReadCloser) (io.ReadSeeker, error) {
	// Read the nonce
	nonceSize := o.gcm.NonceSize()
	nonce := make([]byte, nonceSize)
	_, err := objReader.Read(nonce)
	if err != nil {
		return nil, err
	}

	// Read the rest
	cipherTextBuffer := bytes.NewBuffer(nil)
	_, err = io.Copy(cipherTextBuffer, objReader)
	if err != nil {
		return nil, err
	}

	// Decrypt the data
	plainText, err := o.gcm.Open(nil, nonce, cipherTextBuffer.Bytes(), nil)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(plainText), nil
}
