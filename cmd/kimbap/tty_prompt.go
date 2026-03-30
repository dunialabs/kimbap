package main

import "os"

func isInteractiveTTY(file *os.File) bool {
	if file == nil {
		return false
	}
	fi, err := file.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

var canPromptInTTY = func() bool {
	if outputAsJSON() {
		return false
	}
	return isInteractiveTTY(os.Stdin) && isInteractiveTTY(os.Stdout)
}
