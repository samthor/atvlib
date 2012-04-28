package main

import (
	"log"
	"net"
	"net/http"
)

// Start a webserver serving files from the given path. Take the address given and attach the HTTP
// server to a random port. If portCh is non-nil, sends the chosen port on this channel.
func webserver(path string, addr net.TCPAddr, portCh chan int) {
	addr.Port = 0
	listener, err := net.ListenTCP("tcp", &addr)
	if portCh != nil {
		portCh <- listener.Addr().(*net.TCPAddr).Port
	}

	log.Printf("webserver listening on: %s", listener.Addr())
	err = http.Serve(listener, http.FileServer(http.Dir(path)))
	if err != nil {
		panic(err)
	}
}
