package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/ISBachvarov21/Rune-GO-test/httphandler"
)

func server(srv *http.Server) {
	var handler = &httphandler.HTTPHandler{}

	handler.AddRoute("GET", "/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("ivan e gotin /")
	})

	handler.AddRoute("POST", "/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("POST /")
	})

	srv.Handler = handler
	srv.Addr = "127.0.0.1:8080"
	log.Fatal(srv.ListenAndServe())
}

func main() {
	shutdown := make(chan struct{})
	var srv http.Server

	go func() {
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			if scanner.Text() == "shutdown\n" || scanner.Text() == "shutdown" {
				close(shutdown)
			}
		}
	}()

	go server(&srv)

	for {
		select {
		case <-shutdown:
			srv.Shutdown(context.Background())
		}
	}
}
