package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"log"

	"golang.org/x/crypto/pbkdf2"
)

func generateMeta(filename string, salt []byte) string {
	meta := filename + "\u0000" + base64.StdEncoding.EncodeToString(salt)
	return base64.StdEncoding.EncodeToString([]byte(meta))
}

func parseMeta(meta string) (filename string, salt []byte) {
	decodedMeta, err := base64.StdEncoding.DecodeString(meta)
	if err != nil {
		log.Printf("Error decoding metadata: %v\n", err)
		return
	}

	metaSplit := bytes.Split(decodedMeta, []byte("\u0000"))
	if len(metaSplit) != 2 {
		log.Printf("Invalid metadata\n")
		return
	}

	filename = string(metaSplit[0])
	salt, err = base64.StdEncoding.DecodeString(string(metaSplit[1]))
	if err != nil {
		log.Printf("Error decoding salt: %v\n", err)
		return
	}

	return filename, salt
}

func encryptChunk(chunk []byte, key []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	ciphertext := make([]byte, aes.BlockSize+len(chunk))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], chunk)

	return ciphertext, iv, nil
}

func decrypt(encryptedData []byte, key []byte, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(encryptedData) < aes.BlockSize {
		log.Fatal("ciphertext too short")
	}

	stream := cipher.NewCFBDecrypter(block, iv)
	decryptedData := make([]byte, len(encryptedData)-aes.BlockSize)
	stream.XORKeyStream(decryptedData, encryptedData[aes.BlockSize:])

	return decryptedData, nil
}

func deriveKey(password string) (key []byte, salt []byte) {
	salt = make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		log.Fatal(err)
	}

	key = pbkdf2.Key([]byte(password), salt, 4096, 32, sha256.New)

	return key, salt
}

func deriveSaltedKey(password string, salt []byte) (key []byte) {
	key = pbkdf2.Key([]byte(password), salt, 4096, 32, sha256.New)

	return key
}
