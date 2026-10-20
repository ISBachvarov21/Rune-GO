package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type tokenWriter struct {
	token []byte
	found chan struct{}
	once  sync.Once
}

func (w *tokenWriter) Write(p []byte) (int, error) {
	if bytes.Contains(p, w.token) {
		w.once.Do(func() { close(w.found) })
		return len(p), nil
	}
	os.Stdout.Write(append([]byte("Server: "), p...))
	return len(p), nil
}

type handler struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	mut   sync.Mutex
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
		err := replaceProcess()
		if err != nil {
			log.Println(err)
			return
		}
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Println("watcher.Events closed")
					return
				}

				if event.Has(fsnotify.Write) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) || event.Has(fsnotify.Create) {
					nameSplit := strings.Split(event.Name, ".")
					if nameSplit[len(nameSplit)-1] != "go" {
						continue
					}
					if time.Since(lastEvent).Milliseconds() > 100 {
						log.Println("modified file:", event.Name)
						err = replaceProcess()
						if err != nil {
							log.Println(err)
						}
					}
					lastEvent = time.Now()
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add("./")

	if err != nil {
		log.Fatal(err)
	}

	<-make(chan struct{})
}

func replaceProcess() error {
	currentHandler.mut.Lock()
	defer currentHandler.mut.Unlock()

	executableExtension := ""
	if runtime.GOOS == "windows" {
		executableExtension = ".exe"
	}

	cmd := exec.Command("go", "build", "-o", fmt.Sprintf("./build/main%s", executableExtension), ".")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to build executable: %v", err)
		return err
	}
	compiledFilePath, err := exec.LookPath(fmt.Sprintf("./build/main%s", executableExtension))

	if err != nil {
		log.Printf("Failed to find compiled binary: %v", err)
		return err
	}

	if currentHandler.cmd != nil {
		//log.Println("Cancelling previous handler")
		if currentHandler.stdin != nil {
			if _, err := io.WriteString(currentHandler.stdin, "shutdown\n"); err != nil {
				log.Printf("Failed to send shutdown command: %v", err)
			}
			currentHandler.stdin.Close()
		}

		// Give the process a moment to shut down gracefully, then force-kill.
		done := make(chan error, 1)
		go func() { done <- currentHandler.cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			currentHandler.cmd.Process.Kill()
			<-done
		}
	}

	cmd = exec.Command(compiledFilePath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("Failed to create stdin pipe: %v", err)
		return err
	}

	ready := make(chan struct{})
	tw := &tokenWriter{token: []byte("running\n"), found: ready}
	cmd.Stdout = tw
	cmd.Stderr = tw

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start command: %v", err)
		return err
	}

	currentHandler.cmd = cmd
	currentHandler.stdin = stdin

	if _, err = io.WriteString(stdin, "port:8080\n"); err != nil {
		return err
	}

	select {
	case <-ready:
		log.Print("Server reload succeeded")
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		cmd.Wait()
		return fmt.Errorf("server startup timed out")
	}

	return nil
}
