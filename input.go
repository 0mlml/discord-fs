package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func handleCommand(cmd string) error {
	parts := strings.Split(cmd, " ")

	switch parts[0] {
	case "init":
		intialize()
	case "send":
		if len(parts) != 2 {
			return fmt.Errorf("invalid send command")
		}
		cf, err := chunkFile(parts[1])

		if err != nil {
			return fmt.Errorf("error chunking file: %v", err)
		}

		logger.Printf("Chunked file %s into %d chunks\n", cf.name, len(cf.data))

		return sendChunkedFile(cf)
	case "fetch":
		if len(parts) != 2 {
			return fmt.Errorf("invalid fetch command")
		}

		cf, err := fetchChunkedFile(parts[1])

		if err != nil {
			return fmt.Errorf("error fetching file: %v", err)
		}

		logger.Printf("Decrypting and reconstructing file %s\n", cf.name)

		return reconstructFile(cf, fmt.Sprintf("%s.dec", cf.name))
	}

	return nil
}

func simpleReadPump() {
	reader := bufio.NewReader(os.Stdin)
	for {
		logger.Printf("Enter command: ")
		command, err := reader.ReadString('\n')
		if err != nil {
			logger.Printf("Error reading command: %v\n", err)
			continue
		}

		command = strings.TrimSpace(command)

		if err := handleCommand(command); err != nil {
			logger.Printf("Error handling command \"%s\": %v\n", command, err)
		}
	}
}
