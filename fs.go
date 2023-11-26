package main

import (
	"crypto/aes"
	"fmt"
	"io"
	"os"
)

type chunkedFile struct {
	name string
	salt []byte
	data [][]byte
}

func chunkFile(path string) (f *chunkedFile, err error) {
	var file *os.File
	file, err = os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	key, salt := deriveKey(config.String("your_key"))

	f = &chunkedFile{}

	f.name = file.Name()
	f.salt = salt

	chunkSize := config.Int("max_file_size") - aes.BlockSize

	chunkNumber := 0
	buffer := make([]byte, chunkSize)
	for {
		var bytesRead int
		bytesRead, err = file.Read(buffer)
		if bytesRead > 0 {
			encryptedData, iv, encryptErr := encryptChunk(buffer[:bytesRead], key)
			if encryptErr != nil {
				fmt.Printf("Error encrypting chunk: %v\n", encryptErr)
				return nil, encryptErr
			}

			var chunk []byte
			chunk = append(chunk, iv...)
			chunk = append(chunk, encryptedData...)
			f.data = append(f.data, chunk)

			chunkNumber++
		}
		if err != nil {
			if err == io.EOF {
				break
			}

			fmt.Printf("Error reading file: %v\n", err)
			break
		}
	}

	return f, nil
}

func reconstructFile(f *chunkedFile, outputPath string) error {
	key := deriveSaltedKey(config.String("your_key"), f.salt)

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	for _, chunk := range f.data {
		if len(chunk) < aes.BlockSize {
			return fmt.Errorf("chunk too short, expected at least %d bytes, got %d", aes.BlockSize, len(chunk))
		}

		iv := chunk[:aes.BlockSize]
		encryptedData := chunk[aes.BlockSize:]

		decryptedData, decryptErr := decrypt(encryptedData, key, iv)
		if decryptErr != nil {
			return fmt.Errorf("error decrypting chunk: %v", decryptErr)
		}

		_, writeErr := outputFile.Write(decryptedData)
		if writeErr != nil {
			return fmt.Errorf("error writing to output file: %v", writeErr)
		}
	}

	return nil
}
