package main

import (
	"flag"

	"github.com/ISBachvarov21/Rune-GO/internal"
)

func main() {
	shouldWatch := flag.Bool("watch", false, "Watch for changes and restart the server")

	flag.Parse()

	if *shouldWatch {
		internal.StartWatcher("./routes/")
	} else {
		err := internal.WalkEndpoints("./routes/")
		if err != nil {
			return
		}
	}
}
