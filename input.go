package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unicode"
	"unsafe"
)

func handleCommand(cmd string) error {
	parts := strings.Split(cmd, " ")

	switch parts[0] {
	case "init":
		intialize()
	case "send":
		if len(parts) != 2 {
			return fmt.Errorf("Invalid send command")
		}
		cf, err := chunkFile(parts[1])

		if err != nil {
			return fmt.Errorf("Error chunking file: %v", err)
		}

		logger.Printf("Chunked file %s into %d chunks\n", cf.name, len(cf.data))

		return sendChunkedFile(cf)
	case "fetch":
		if len(parts) != 2 {
			return fmt.Errorf("Invalid fetch command")
		}

		cf, err := fetchChunkedFile(parts[1])

		if err != nil {
			return fmt.Errorf("Error fetching file: %v", err)
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

func enableRawMode(fd int) (*syscall.Termios, error) {
	var oldState syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), syscall.TCGETS, uintptr(unsafe.Pointer(&oldState)), 0, 0, 0)
	if err != 0 {
		return nil, err
	}

	newState := oldState
	newState.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG
	newState.Iflag &^= syscall.IXON | syscall.ICRNL
	newState.Cc[syscall.VMIN] = 1
	newState.Cc[syscall.VTIME] = 0

	_, _, err = syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), syscall.TCSETS, uintptr(unsafe.Pointer(&newState)), 0, 0, 0)
	if err != 0 {
		return nil, err
	}

	return &oldState, nil
}

func disableRawMode(fd int, state *syscall.Termios) error {
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), syscall.TCSETS, uintptr(unsafe.Pointer(state)), 0, 0, 0)
	return err
}

var (
	tabCompletionIndex   = -1
	tabCompletionOptions []string
)

func readInput() (string, error) {
	fd := int(os.Stdin.Fd())

	oldState, err := enableRawMode(fd)
	if err != nil {
		return "", err
	}
	defer disableRawMode(fd, oldState)

	var buf [1]byte
	var b strings.Builder

	cursorPos := 0

	for {
		n, err := syscall.Read(fd, buf[:])
		if err != nil {
			return "", err
		}
		if n <= 0 {
			continue
		}

		c := buf[0]

		if c != '\t' {
			tabCompletionIndex = -1
		}

		switch {
		case c == '\n' || c == '\r':
			logger.Printf("\n")
			return b.String(), nil
		case c == '\x1b':
			nextBytes := make([]byte, 2)
			if n, err := syscall.Read(fd, nextBytes); n == 2 && err == nil {
				if nextBytes[0] == '[' {
					switch nextBytes[1] {
					case 'C':
						if cursorPos < len(b.String()) {
							cursorPos++
							updateDisplay(&b, cursorPos)
						}
					case 'D':
						if cursorPos > 0 {
							cursorPos--
							updateDisplay(&b, cursorPos)
						}
					}
				}
			}
		case c == '\x03':
			os.Exit(0)
		case c == '\t':
			parts := strings.Split(b.String(), " ")
			if tabCompletionIndex == -1 {
				tabCompletionOptions, err = handleTabCompletion(parts)

				if err != nil {
					continue
				}

				tabCompletionIndex = 0
			}

			if len(tabCompletionOptions) == 0 {
				continue
			}

			parts[len(parts)-1] = tabCompletionOptions[tabCompletionIndex]

			b.Reset()
			b.WriteString(strings.Join(parts, " "))
			cursorPos = len(b.String())
			updateDisplay(&b, cursorPos)

			tabCompletionIndex = (tabCompletionIndex + 1) % len(tabCompletionOptions)
		case unicode.IsPrint(rune(c)):
			currentStr := b.String()
			b.Reset()
			b.WriteString(currentStr[:cursorPos] + string(c) + currentStr[cursorPos:])
			cursorPos++
			updateDisplay(&b, cursorPos)
		case c == 127:
			if cursorPos > 0 {
				currentStr := b.String()
				b.Reset()
				b.WriteString(currentStr[:cursorPos-1] + currentStr[cursorPos:])
				cursorPos--
				updateDisplay(&b, cursorPos)
			}
		}
	}
}

func updateDisplay(b *strings.Builder, cursorPos int) {
	logger.Printf("\033[2K\033[G")

	logger.Printf("Enter command: %s", b.String())

	if cursorPos < len(b.String()) {
		for i := 0; i < len(b.String())-cursorPos; i++ {
			logger.Printf("\033[D")
		}
	}
}

func handleTabCompletion(parts []string) ([]string, error) {
	options := make([]string, 0)
	search := parts[len(parts)-1]

	switch len(parts) {
	case 1:
		for _, command := range []string{"init", "send", "fetch"} {
			if strings.HasPrefix(command, search) {
				options = append(options, command)
			}
		}

		return options, nil
	case 2:
		switch parts[0] {
		case "send":
			path := "."
			if strings.Contains(search, "/") {
				parts := strings.Split(search, "/")
				search = parts[len(parts)-1]
				path = strings.Join(parts[:len(parts)-1], "/")
			}

			files, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}

			for _, file := range files {
				if strings.HasPrefix(file.Name(), search) {
					if file.IsDir() {
						options = append(options, fmt.Sprintf("%s/%s/", path, file.Name()))
					} else {
						options = append(options, fmt.Sprintf("%s/%s", path, file.Name()))
					}
				}
			}

			return options, nil
		case "fetch":
			for _, id := range idHistory {
				if strings.HasPrefix(id, search) {
					options = append(options, id)
				}
			}

			return options, nil
		}
	}
	return options, nil
}

func advancedReadPump() {
	for {
		logger.Flush()
		logger.Printf("Enter command: ")
		command, err := readInput()
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
