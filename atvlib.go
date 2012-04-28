package atvlib

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"errors"
	"io"
)

const USERAGENT = "MediaControl/1.0"

type AppleTVLink struct {
	cn *net.TCPConn
	r  *bufio.Reader
}

func NewAppleTVLink(host string) (m *AppleTVLink, err error) {
	cn, err := net.Dial("tcp", host)
	if err != nil {
		return
	}
	m = &AppleTVLink{cn.(*net.TCPConn), bufio.NewReader(cn)}

	// Perform the initial 'POST /reverse HTTP/1.1'...
	h := make(http.Header)
	h.Add("Upgrade", "PTTH/1.0")
	h.Add("Connection", "Upgrade")
	err = m.Do("/reverse", h, nil)
	if err != nil {
		m.Close()
	}
	return
}

func (m *AppleTVLink) Do(target string, header http.Header, content []byte) error {
	log.Printf("control req: %s => %s (clen=%d)", target, header, len(content))
	_, err := fmt.Fprintf(m.cn, fmt.Sprintf("POST %s HTTP/1.1\r\n", target))
	if err != nil {
		return err
	}

	err = header.Write(m.cn)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(m.cn, "Content-Length: %d\r\nUser-Agent: %s\r\n\r\n", len(content), USERAGENT)
	if err == nil && content != nil {
		_, err = fmt.Fprintf(m.cn, "%s", content)
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
			if header.Get("Upgrade") != "" && strings.HasPrefix(string(line), "HTTP/1.1 101") {
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

func (m *AppleTVLink) Idle() {
	_, _, err := m.r.ReadLine()
	if err != io.EOF {
		panic(err)
	}
	log.Printf("control EOF, idle done")
}

func (m *AppleTVLink) LocalAddr() net.TCPAddr {
	ptr := m.cn.LocalAddr().(*net.TCPAddr)
	addr := *ptr
	addr.Port = 0
	return addr
}

func (m *AppleTVLink) Close() {
	m.cn.Close()
}
