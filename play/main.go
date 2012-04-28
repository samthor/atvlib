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

var path = flag.String("path", "", "path to .mp4 (or other valid format) to play")
var target = flag.String("target", "apple-tv.local:7000", "appletv to connect to")

func main() {
	flag.Parse()
	if *path == "" {
		log.Fatalf("fatal; must specify -path")
	}

	// Create a temp folder, and symlink to our target.
	// TODO: If this is killed with ctrl-c, which is usual behaviour, then the deferred
	// remove may not occur. Perhaps we need to intercept signals (but this is weird).
	servepath, _ := ioutil.TempDir("", "aplay")
	filepath, _ := filepath.Abs(*path)
	err := os.Symlink(filepath, servepath+SUFFIX)
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

	// Send PLAY.
	data := fmt.Sprintf("Content-Location: %s\r\nStart-Position: 0\r\n", address)
	err = link.Do("/play", nil, []byte(data))
	if err != nil {
		panic(err)
	}

	// Wait for the Apple TV to EOF!
	link.Idle()
}
