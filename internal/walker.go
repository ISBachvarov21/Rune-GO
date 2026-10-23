package internal

import (
	"bytes"
	"embed"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

type Endpoint struct {
	Method  string
	Path    string
	Handler string
}

var endpointRxp = regexp.MustCompile(`(?i)(?P<DECLARATOR>RUNE) (?P<METHOD>GET|POST|OPTIONS|HEAD|PUT|PATCH|DELETE) (?P<PATH>\S*)`)
var endpointList []Endpoint
var packageName string

//go:embed templates/registerroutes.go.tmpl
var templateFS embed.FS

/*
walk through given directory and detect any endpoint functions with the following signature:

// RUNE VERB(PATH)

func ...(...) ... {}
*/
func WalkEndpoints(dir string) error {
	var files []string
	endpointList = []Endpoint{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			if info.Name() == "routes" {
				return nil
			}
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

		// Add the file to the list
		files = append(files, path)

		return nil
	})

	if err != nil {
		return err
	}

	for _, file := range files {
		if err := walkAST(file); err != nil {
			fmt.Printf("Error walking AST for file %s: %v\n", file, err)
		}
	}

	f, err := os.OpenFile(filepath.Join(dir, "registerroutes.go"), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	defer f.Close()

	tmpl, err := template.ParseFS(templateFS, "templates/registerroutes.go.tmpl")
	if err != nil {
		return err
	}

	buffer := bytes.NewBuffer([]byte{})

	err = tmpl.Execute(buffer, struct {
		PackageName string
		Routes      []Endpoint
	}{
		PackageName: packageName,
		Routes:      endpointList,
	})
	if err != nil {
		return err
	}

	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		return err
	}

	stat, _ := f.Stat()
	originalContents := make([]byte, stat.Size())

	f.Seek(0, 0)
	_, err = f.Read(originalContents)
	if err != nil && err.Error() != "EOF" {
		return err
	}

	if bytes.Equal(originalContents, formatted) {
		fmt.Println("No changes needed")
		return nil // No changes needed
	}

	err = f.Truncate(0)
	if err != nil {
		return err
	}

	_, err = f.Write(formatted)
	if err != nil {
		return err
	}

	return nil
}

// Check if the file contains the RUNE VERB(PATH) comment
func containsRuneSignature(data []byte) bool {
	return endpointRxp.MatchString(string(data))
}

func walkAST(filePath string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)

	if err != nil {
		return fmt.Errorf("failed to parse %s: %v", filePath, err)
	}

	packageName = f.Name.Name
	ast.Inspect(f, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok && fn.Doc != nil {
			for _, comment := range fn.Doc.List {
				if endpointRxp.MatchString(comment.Text) {
					matches := endpointRxp.FindStringSubmatch(comment.Text)
					methodIndex := endpointRxp.SubexpIndex("METHOD")
					pathIndex := endpointRxp.SubexpIndex("PATH")
					method := matches[methodIndex]
					path := matches[pathIndex]
					endpointList = append(endpointList, Endpoint{Method: strings.ToUpper(method), Path: path, Handler: fn.Name.Name})
				}
			}
			for _, arg := range fn.Type.Params.List {
				for _, name := range arg.Names {
					fmt.Printf("Found argument: %s\n", name.Name)
				}
				//if len(arg.Names) > 0 {
				//	fmt.Printf("Found argument: %s\n", arg.Names[0].Name)
				//}
			}
		}
		return true
	})

	return nil
}
