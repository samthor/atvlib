package main

import (
	"flag"
	"fmt"
	"github.com/samthor/atvlib"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

const SUFFIX = "/atv.mp4"

var target = flag.String("target", "apple-tv.local:7000", "appletv to connect to")

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatalf("fatal; expected filename")
	}
	path := flag.Args()[0]

	// Create a temp folder, and symlink to our target.
	// TODO: If this is killed with ctrl-c, which is usual behaviour, then the deferred
	// remove may not occur. Perhaps we need to intercept signals (but this is weird).
	servepath, _ := ioutil.TempDir("", "aplay")
	path, _ = filepath.Abs(path)
	err := os.Symlink(path, servepath+SUFFIX)
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(servepath)
	log.Printf("created serving symlink: %s", servepath+SUFFIX)

	// Create AppleTVLink.
	link, err := atvlib.NewAppleTVLink(*target)
	if err != nil {
		panic(err)
	}

	// Start webserver, grab its local port.
	local := link.LocalAddr()
	portCh := make(chan int)
	go webserver(servepath, local, portCh)
	local.Port = <-portCh

	// Extract complete URL of video.
	address := fmt.Sprintf("http://%s%s", local.String(), SUFFIX)
	log.Printf("serving at: %s", address)

	// Play the file that we're now serving.
	err = link.DoPlay(address)
	if err != nil {
		panic(err)
	}

	// Wait for the Apple TV to EOF!
	link.Idle()
}
