package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func confirmShortcutSetup(serviceName string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	for {
		_, _ = fmt.Fprintf(os.Stdout, "Configure default shortcut aliases for %s now? [Y/n] ", serviceName)
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) {
				return true, nil
			}
			if len(line) == 0 {
				return false, err
			}
		}
		answer := strings.ToLower(strings.TrimSpace(line))
		switch answer {
		case "", "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			_, _ = fmt.Fprintln(os.Stdout, "Please answer y or n.")
		}
	}
}
