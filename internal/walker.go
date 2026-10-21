package internal

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
)

var endpointRxp = regexp.MustCompile(`(?i)(?P<DECLARATOR>RUNE) (?P<METHOD>GET|UPDATE|PUT|DELETE) (?P<PATH>/.*)`)

/*
walk through given directory and detect any endpoint functions with the following signature:

// RUNE VERB(PATH)

func ...(...) ... {}
*/
func WalkEndpoints(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return fs.SkipDir
		}

		// Skip non-Go files
		if filepath.Ext(path) != ".go" {
			return nil
		}

		// Load the file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Check if the file contains the RUNE VERB(PATH) comment
		if !containsRuneSignature(data) {
			return nil
		}

		// If it does, print the file path
		println("Found rune signature in file:", path)

		return nil
	})
}

// Check if the file contains the RUNE VERB(PATH) comment
func containsRuneSignature(data []byte) bool {
	return endpointRxp.MatchString(string(data))
}
