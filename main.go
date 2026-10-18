package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type handler struct {
	cmd  *exec.Cmd
	pipe *io.WriteCloser
	mut  sync.Mutex
}

var currentHandler handler

func main() {
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		log.Fatal(err)
		return
	}

	defer watcher.Close()

	lastEvent := time.Now()
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Println("watcher.Events closed")
					return
				}

				if event.Has(fsnotify.Write | fsnotify.Create) {
					nameSplit := strings.Split(event.Name, ".")
					if nameSplit[len(nameSplit)-1] != "go" {
						continue
					}
					if time.Since(lastEvent).Milliseconds() > 1000 {
						lastEvent = time.Now()
						log.Println("modified file:", event.Name)
						err = runFile(strings.ReplaceAll(fmt.Sprintf(".\\%v", event.Name), "\\", "/"))
						if err != nil {
							log.Println(err)
						}
					}
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add("./test/")

	if err != nil {
		log.Fatal(err)
	}

	<-make(chan struct{}, 0)
}

func runFile(filePath string) error {
	currentHandler.mut.Lock()
	defer currentHandler.mut.Unlock()

	dir, file := filepath.Split(filePath)
	cmd := exec.Command("go", "run", file)
	cmd.Dir = dir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Println("running", filePath)

	if err := cmd.Start(); err != nil {
		err = cmd.Cancel()
		if err != nil {
			return err
		}

		return err
	}

	if currentHandler.cmd != nil {
		log.Println("cancelling previous handler")
		if _, err := io.WriteString(*currentHandler.pipe, "shutdown\n"); err != nil {
			return err
		}

		if err := (*currentHandler.pipe).Close(); err != nil {
			return err
		}
	}

	currentHandler.cmd = cmd
	currentHandler.pipe = &stdin

	return nil
}
