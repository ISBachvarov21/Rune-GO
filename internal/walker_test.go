package internal

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWalkEndpoints_TableDriven(t *testing.T) {
	tests := []struct {
		name             string
		goFileContent    string
		expectErr        bool
		expectedFile     string // The file we expect the generator to output
		expectedContents string // Token/String fragments we expect inside the generated file
	}{
		{
			name: "Valid GET Route with Context Only",
			goFileContent: `package routes
import "://github.com"

// RUNE GET /
func GetRoot(c *runehttp.Context) (runehttp.Response, error) {
	return runehttp.Text("Hello World!"), nil
}`,
			expectErr:    false,
			expectedFile: "registerroutes.go",
			expectedContents: `handler.AddRoute(
		"GET",
		"/",
		GetRoot,
	)`,
		},
		{
			name: "Valid POST Route with Serializable Struct",
			goFileContent: `package routes
import "://github.com"

type Serializable struct {
	Field string ` + "`" + `json:"sf"` + "`" + `
}

// RUNE POST /serializable
func PostSerializable(s Serializable) (runehttp.Response, error) {
	return runehttp.OK(nil), nil
}`,
			expectErr:    false,
			expectedFile: "registerroutes.go",
			expectedContents: `handler.AddRoute(
		"POST",
		"/serializable",
		PostSerializable,
	)`,
		},
		{
			name: "Valid Route with Multiple Arguments (Context and Struct)",
			goFileContent: `package routes
import "://github.com"

type PostIM struct { Title string }

// RUNE POST /post
func CreatePost(c *runehttp.Context, p PostIM) (runehttp.Response, error) {
	return runehttp.OK(nil), nil
}`,
			expectErr:    false,
			expectedFile: "registerroutes.go",
			expectedContents: `handler.AddRoute(
		"POST",
		"/post",
		CreatePost,
	)`,
		},
		{
			name: "Embedded Comment Directive Placement",
			goFileContent: `package routes
import "://github.com"

// Some random documentation context
// Hey look! RUNE PUT /update/status here in the middle
// More docs
func UpdateStatus(c *runehttp.Context) (runehttp.Response, error) {
	return runehttp.OK(nil), nil
}`,
			expectErr:    false,
			expectedFile: "registerroutes.go",
			expectedContents: `handler.AddRoute(
		"PUT",
		"/update/status",
		UpdateStatus,
	)`,
		},
		{
			name: "Function without RUNE directive is Ignored",
			goFileContent: `package routes
import "://github.com"

// Just a standard helper function
func HelperFunction() string {
	return "nothing to see here"
}`,
			expectErr:    false,
			expectedFile: "registerroutes.go",
			// It shouldn't contain HelperFunction registration block
			expectedContents: "",
		},
		{
			name: "Function with RUNE directive deep inside a comment is NOT Ignored",
			goFileContent: `package routes
import "://github.com"
// RUNE GET /
func GetRoot(c *runehttp.Context) (runehttp.Response, error) {
	return runehttp.Text("Hello World!"), nil
}
// Some other comment
// another comment, this time before directive RUNE POST /deep/inside some more comments
// and some more comments
func PostDeep(c *runehttp.Context) (runehttp.Response, error) {
	return runehttp.OK(nil), nil
}`,
			expectErr:    false,
			expectedFile: "registerroutes.go",
			expectedContents: `handler.AddRoute(
		"POST",
		"/deep/inside",
		PostDeep,
	)`,
		},
		{
			name: "Invalid arguments in function",
			goFileContent: `package routes
import "://github.com"

type Serializable struct {
	Field string ` + "`json:" + `"field"` + "`" + `
}

// RUNE POST /path
func InvalidRoute(s Serializable, *runehttp.Context) (runehttp.Response, error) {
	return runehttp.OK(nil), nil
}
`,
			expectErr:        true,
			expectedFile:     "",
			expectedContents: "",
		},
		{
			name: "Function with unnamed context pointer arg",
			goFileContent: `package routes
import "://github.com"

// RUNE GET /
func UnnamedContext(*runehttp.Context) (runehttp.Response, error) {
	return runehttp.OK(nil), nil
}
`,
			expectErr:    false,
			expectedFile: "registerroutes.go",
			expectedContents: `handler.AddRoute(
		"GET",
		"/",
		UnnamedContext,
	)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Setup isolated environment
			dir := t.TempDir()

			goFilePath := filepath.Join(dir, "routetest.go")
			err := os.WriteFile(goFilePath, []byte(strings.TrimSpace(tt.goFileContent)), 0644)
			if err != nil {
				t.Fatalf("Setup Failed: couldn't write temp go file: %v", err)
			}

			// 2. Execute target framework code walker
			err = WalkEndpoints(dir)

			// 3. Evaluate error return expectations
			if (err != nil) != tt.expectErr {
				t.Fatalf("WalkEndpoints() returned error = %v, expectErr = %v", err, tt.expectErr)
			}

			// If error was expected or no generation file output target is set, exit case early
			if tt.expectErr || tt.expectedFile == "" {
				return
			}

			// 4. Validate output file generation side effects
			genFilePath := filepath.Join(dir, tt.expectedFile)
			generatedBytes, err := os.ReadFile(genFilePath)
			if err != nil {
				// If we didn't expect any registrations, file might not be created or might be empty
				if tt.expectedContents == "" {
					return
				}
				t.Fatalf("Validation Failed: expected file %s was not generated: %v", tt.expectedFile, err)
			}

			// 5. Code formatting & content checks
			formattedGen, formatErr := format.Source(generatedBytes)
			if formatErr != nil {
				t.Fatalf("Warning: Generated code failed gofmt formatting: %v", formatErr)
			}

			genStr := string(formattedGen)

			if tt.expectedContents != "" {
				// Clean formatting structures for predictable substring matching
				cleanGen := stripWhitespace(genStr)
				cleanExpected := stripWhitespace(tt.expectedContents)

				if !strings.Contains(cleanGen, cleanExpected) {
					t.Fatalf("Assertion Failure: Generated route code missing expected registrations.\n\n[WANTED SUBSTRING]:\n%s\n\n[GOT FULL GENERATED FILE]:\n%s", tt.expectedContents, genStr)
				}
			} else {
				// If expectedContents is explicitly empty string, ensure nothing was registered
				if strings.Contains(genStr, "AddRoute") {
					t.Fatalf("Assertion Failure: Expected empty registration file, but found mappings:\n%s", genStr)
				}
			}
		})
	}
}

func TestMultifileEndpoints(t *testing.T) {
	tests := []struct {
		name             string
		goFileContents   []string
		expectErr        bool
		expectedErr      string
		expectedFile     string   // The file we expect the generator to output
		expectedContents []string // Token/String fragments we expect inside the generated file
	}{
		{
			name: "Multifile package name mismatch",
			goFileContents: []string{
				`package package1

import "://github.com"

// RUNE GET /
func Root() (runehttp.Response, error) {
	return runehttp.OK(nil), nil
}`,

				`package package2

import "://github.com"

// RUNE GET /
func Root2() (runehttp.Response, error) {
	return runehttp.OK(nil), nil
}`,
			},
			expectErr:        true,
			expectedErr:      "package name mismatch",
			expectedFile:     "",
			expectedContents: nil,
		},
		{
			name: "Multifile different endpoint registration",
			goFileContents: []string{
				`package routes

import "://github.com"

// RUNE GET /
func Root() (runehttp.Response, error) {
	return runehttp.OK(nil), nil
}`,
				`package routes

import "://github.com"

// RUNE POST /
func PostRoot() (runehttp.Response, error) {
	return runehttp.OK(nil), nil
}`,
			},
			expectErr:    false,
			expectedErr:  "",
			expectedFile: "registerroutes.go",
			expectedContents: []string{
				`handler.AddRoute(
		"GET",
		"/",
		Root,
	)`,
				`handler.AddRoute(
		"POST",
		"/",
		PostRoot,
	)`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Setup isolated environment
			dir := t.TempDir()

			for i, fileContent := range tt.goFileContents {
				goFilePath := filepath.Join(dir, fmt.Sprintf("routetest%d.go", i))
				err := os.WriteFile(goFilePath, []byte(strings.TrimSpace(fileContent)), 0644)
				if err != nil {
					t.Fatalf("Setup Failed: couldn't write temp go file: %v", err)
				}
			}

			// 2. Execute target framework code walker
			err := WalkEndpoints(dir)

			// 3. Evaluate error return expectations
			if (err != nil) != tt.expectErr {
				t.Fatalf("WalkEndpoints() returned error = %v, expectErr = %v", err, tt.expectErr)
			}

			// If error was expected or no generation file output target is set, exit case early
			if tt.expectErr || tt.expectedFile == "" {
				if tt.expectErr && err != nil && !strings.Contains(err.Error(), tt.expectedErr) {
					t.Fatalf("WalkEndpoints() returned error = %v, expected error = %v", err, tt.expectedErr)
				}
				return
			}

			// 4. Validate output file generation side effects
			genFilePath := filepath.Join(dir, tt.expectedFile)
			generatedBytes, err := os.ReadFile(genFilePath)
			if err != nil {
				// If we didn't expect any registrations, file might not be created or might be empty
				if tt.expectedContents == nil {
					return
				}
				t.Fatalf("Validation Failed: expected file %s was not generated: %v", tt.expectedFile, err)
			}

			// 5. Code formatting & content checks
			formattedGen, formatErr := format.Source(generatedBytes)
			if formatErr != nil {
				t.Fatalf("Warning: Generated code failed gofmt formatting: %v", formatErr)
			}

			genStr := string(formattedGen)

			if tt.expectedContents != nil {
				for _, expectation := range tt.expectedContents {
					// Clean formatting structures for predictable substring matching
					cleanGen := stripWhitespace(genStr)
					cleanExpected := stripWhitespace(expectation)

					if expectation != "" {
						if !strings.Contains(cleanGen, cleanExpected) {
							t.Fatalf("Assertion Failure: Generated route code missing expected registrations.\n\n[WANTED SUBSTRING]:\n%s\n\n[GOT FULL GENERATED FILE]:\n%s", expectation, genStr)
						}
					} else {
						// If expectation is explicitly empty string, ensure nothing was registered
						if strings.Contains(genStr, "AddRoute") {
							t.Fatalf("Assertion Failure: Expected empty registration file, but found mappings:\n%s", genStr)
						}
					}
				}
			}
		})
	}
}

func TestStaleCodeGeneration(t *testing.T) {
	goFileContent := `package routes

import "://github.com"

// RUNE GET /
func Root() (runehttp.Response, error) {
	return runehttp.OK(nil), nil
}
`
	expectedFile := "registerroutes.go"
	expectedContentBeforeDeletion := `handler.AddRoute(
		"GET",
		"/",
		Root,
	)`

	expectedContentAfterDeletion := `func RegisterRoutes(handler *runehttp.HTTPHandler) error {
	return nil
}`
	dir := t.TempDir()

	goFilePath := filepath.Join(dir, "routetest.go")
	err := os.WriteFile(goFilePath, []byte(strings.TrimSpace(goFileContent)), 0644)
	if err != nil {
		t.Fatalf("Setup Failed: couldn't write temp go file: %v", err)
	}

	err = WalkEndpoints(dir)
	if err != nil {
		t.Fatalf("WalkEndpoints() returned error = %v", err)
	}

	genFilePath := filepath.Join(dir, expectedFile)
	generatedBytes, err := os.ReadFile(genFilePath)
	if err != nil {
		// If we didn't expect any registrations, file might not be created or might be empty
		if expectedContentBeforeDeletion == "" {
			return
		}
		t.Fatalf("Validation Failed: expected file %s was not generated: %v", expectedFile, err)
	}

	formattedGen, formatErr := format.Source(generatedBytes)
	if formatErr != nil {
		t.Fatalf("Warning: Generated code failed gofmt formatting: %v", formatErr)
	}

	genStr := string(formattedGen)

	if !strings.Contains(genStr, expectedContentBeforeDeletion) {
		t.Fatalf("Assertion Failure: Expected stale code generation before deletion to be detected, but it wasn't:\n%s", genStr)
	}

	os.Remove(goFilePath)

	err = WalkEndpoints(dir)
	if err != nil {
		t.Fatalf("WalkEndpoints() returned error = %v", err)
	}

	generatedBytes, err = os.ReadFile(genFilePath)
	if err != nil {
		t.Fatalf("Validation Failed: expected file %s did not get generated: %v", expectedFile, err)
	}

	formattedGen, formatErr = format.Source(generatedBytes)
	if formatErr != nil {
		t.Fatalf("Warning: Generated code failed gofmt formatting: %v", formatErr)
	}

	genStr = string(formattedGen)
	if !strings.Contains(genStr, expectedContentAfterDeletion) {
		t.Fatalf("Assertion Failure: Expected stale code generation after deletion to be detected, but it wasn't:\n%s", genStr)
	}
}

// Helper utility to make AST code substring validation whitespace-insensitive
func stripWhitespace(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s, " ", ""), "\t", ""), "\n", "")
}
