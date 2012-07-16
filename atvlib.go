package atvlib

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const USERAGENT = "MediaControl/1.0"

type AppleTVOp struct {
	Target  string
	Header  http.Header
	Content []byte
}

type pair struct {
	op    AppleTVOp
	errCh chan error
}

type AppleTVLink struct {
	cn *net.TCPConn
	r  *bufio.Reader
	op chan pair
}

func NewAppleTVLink(host string) (m *AppleTVLink, err error) {
	cn, err := net.Dial("tcp", host)
	if err != nil {
		return
	}
	m = &AppleTVLink{
		cn: cn.(*net.TCPConn),
		r:  bufio.NewReader(cn),
		op: make(chan pair),
	}
	go m.run()

	// Perform the initial 'POST /reverse HTTP/1.1'...

	h := make(http.Header)
	h.Add("Upgrade", "PTTH/1.0")
	h.Add("Connection", "Upgrade")
	err = m.Do(&AppleTVOp{Target: "/reverse", Header: h})
	if err != nil {
		m.Close()
	}
	return
}

// run is the background goroutine for this AppleTVLink. It accepts control requests and writes
// noop operations to the device in the background.
func (m *AppleTVLink) run() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.perform(&AppleTVOp{Target: "/noop"})
		case p, ok := <-m.op:
			if !ok {
				log.Printf("op channel shutdown; leaving run")
				return
			}
			log.Printf("got pair: %s", p)
			err := m.perform(&p.op)
			p.errCh <- err
		}
	}
}

func (m *AppleTVLink) Do(op *AppleTVOp) error {
	ch := make(chan error)
	m.op <- pair{*op, ch}
	return <-ch
}

// perform calls the device with a request. Target is in the form "/play", "/cmd", etc. Header
// is optional, and if Content is specified, this request is made via POST.
// This method returns an error if the return status is not in the 2xx range.
func (m *AppleTVLink) perform(op *AppleTVOp) error {
	log.Printf("control req: %s => %s (clen=%d)", op.Target, op.Header, len(op.Content))
	_, err := fmt.Fprintf(m.cn, fmt.Sprintf("POST %s HTTP/1.1\r\n", op.Target))
	if err != nil {
		return err
	}

	err = op.Header.Write(m.cn)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(m.cn, "Content-Length: %d\r\nUser-Agent: %s\r\n\r\n", len(op.Content), USERAGENT)
	if err == nil && op.Content != nil {
		_, err = fmt.Fprintf(m.cn, "%s", op.Content)
	}
	if err != nil {
		return err
	}

	// Wait for a valid response.
	first := true
	for {
		line, prefix, err := m.r.ReadLine()
		if err != nil {
			return err
		}
		if prefix == true {
			return errors.New("ReadLine() returned prefix=true, unhandled")
		}
		if first {
			if op.Header.Get("Upgrade") != "" && strings.HasPrefix(string(line), "HTTP/1.1 101") {
				// This is a hack for our intial Upgrade call. This is fine.
			} else if !strings.HasPrefix(string(line), "HTTP/1.1 2") {
				return errors.New(string(line))
			}
			log.Printf("req OK: %s", line)
			first = false
		}

		if len(line) == 0 {
			if first == true {
				return errors.New("unexpected HTTP response; no status line")
			}
			return nil
		}
	}
	panic("should not get here")
}

// DoPlay asks the Apple TV to play the content at the given address.
func (m *AppleTVLink) DoPlay(address string) error {
	data := fmt.Sprintf("Content-Location: %s\r\nStart-Position: 0\r\n", address)
	return m.Do(&AppleTVOp{
		Target:  "/play",
		Content: []byte(data),
	})
}

// Idle waits until the HTTP connection to the Apple TV causes an EOF.
func (m *AppleTVLink) Idle() {
	// TODO: not useful as a method
	_, _, err := m.r.ReadLine()
	if err != io.EOF {
		panic(err)
	}
	log.Printf("control EOF, idle done")
}

// Return the net.TCPAddr of the local-end of this link. Useful for finding the local address the
// device could dial back to, for e.g., serving media via HTTP.
func (m *AppleTVLink) LocalAddr() net.TCPAddr {
	ptr := m.cn.LocalAddr().(*net.TCPAddr)
	addr := *ptr
	addr.Port = 0
	return addr
}

// Close this link.
func (m *AppleTVLink) Close() {
	if m.op == nil {
		panic("AppleTVLink already closed")
	}
	close(m.op)
	m.op = nil
	m.cn.Close()
}
