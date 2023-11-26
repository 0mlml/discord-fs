package main

import (
	"fmt"
	"sync"
)

type Logger struct {
	persistentLines map[string]string
	orderedKeys     []string
	lastUpdateLines int
	mu              sync.Mutex
}

func ansiMoveUp() {
	fmt.Printf("\033[1A")
}

func ansiCleanUp(n int) {
	for i := 0; i < n; i++ {
		ansiMoveUp()
		ansiClearLine()
	}
}

func ansiClearLine() {
	fmt.Printf("\033[2K\r")
}

func NewLogger() *Logger {
	return &Logger{
		persistentLines: make(map[string]string),
		orderedKeys:     make([]string, 0),
		lastUpdateLines: 0,
	}
}

func (l *Logger) AddLine(key, value string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.persistentLines[key]; !exists {
		l.orderedKeys = append(l.orderedKeys, key)
	}
	l.persistentLines[key] = value

	l.updateDisplay()
}

func (l *Logger) GetLine(key string) string {
	return l.persistentLines[key]
}

func (l *Logger) RemoveLine(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.persistentLines, key)
	for i, k := range l.orderedKeys {
		if k == key {
			l.orderedKeys = append(l.orderedKeys[:i], l.orderedKeys[i+1:]...)
			break
		}
	}

	l.updateDisplay()
}

func (l *Logger) Printf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.lastUpdateLines > 0 {
		ansiCleanUp(l.lastUpdateLines)
	}

	fmt.Printf(format, args...)
	l.lastUpdateLines = 0

	l.updateDisplay()
}

func (l *Logger) updateDisplay() {
	if l.lastUpdateLines > 0 {
		ansiCleanUp(l.lastUpdateLines)
	}

	for _, key := range l.orderedKeys {
		fmt.Println(l.persistentLines[key])
	}

	l.lastUpdateLines = len(l.orderedKeys)
}

func (l *Logger) ClearAll() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.persistentLines = make(map[string]string)
	l.orderedKeys = []string{}
	l.lastUpdateLines = 0
}

func (l *Logger) Flush() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.updateDisplay()
}

func ProgressBarUtil(current int, total int) string {
	percentage := float64(current) / float64(total)
	progressBar := "["
	for i := 0; i < 10; i++ {
		if float64(i)/10.0 < percentage {
			progressBar += "#"
		} else {
			progressBar += " "
		}
	}
	progressBar += "]"
	return progressBar
}
