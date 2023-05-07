package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
)

var (
	key = []byte("5a5c9f7f6da302d2c82f23e34159434d")
)

func AesEncrypt(plainText []byte) (data []byte, err error) {
	// check if key length is valid
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.New("invalid key length")
	}

	// create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// generate random IV
	iv := make([]byte, aes.BlockSize)
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// encrypt plaintext
	ciphertext := make([]byte, len(plainText))
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext, plainText)

	// append IV to ciphertext
	ciphertext = append(iv, ciphertext...)

	//base64 encode ciphertext
	encodedCiphertext := make([]byte, base64.StdEncoding.EncodedLen(len(ciphertext)))
	base64.StdEncoding.Encode(encodedCiphertext, ciphertext)
	st := hex.EncodeToString(encodedCiphertext)
	return []byte(st), nil
}

func AesDecrypt(cipherText []byte) ([]byte, error) {
	st, err := hex.DecodeString(string(cipherText))
	if err != nil {
		return nil, err
	}
	// check if key length is valid
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.New("invalid key length")
	}

	// base64 decode ciphertext
	decodedCiphertext := make([]byte, base64.StdEncoding.DecodedLen(len(st)))
	n, err := base64.StdEncoding.Decode(decodedCiphertext, st)
	if err != nil {
		return nil, err
	}
	decodedCiphertext = decodedCiphertext[:n]

	// make sure the decoded ciphertext is long enough
	if len(decodedCiphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	// extract the IV from the ciphertext
	iv := decodedCiphertext[:aes.BlockSize]
	decodedCiphertext = decodedCiphertext[aes.BlockSize:]

	// create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// create stream cipher
	stream := cipher.NewCTR(block, iv)

	// decrypt ciphertext
	plainText := make([]byte, len(decodedCiphertext))
	stream.XORKeyStream(plainText, decodedCiphertext)

	return plainText, nil
}
