package main

import (
	// "bufio"
	// "fmt"

	"github.com/ilius/crock32"
	// "github.com/jpillora/backoff"
	"github.com/multiformats/go-multihash"

	"net"
	// "os/exec"
	// "path/filepath"
	// "regexp"
	"strconv"
	"strings"
	"time"
)

func ToTimestamp(d time.Time) int64 {
	return d.UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

// FreePort is
func FreePort() int {
	l, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// CurrentTime is
func CurrentTime() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

// ToPeerID returns b58-encoded ID. it converts to b58 if b32-encoded.
func ToPeerID(s string) string {
	m, err := multihash.FromB58String(s)
	if err == nil {
		return m.B58String()
	}

	c, err := crock32.Decode(s)
	if err != nil {
		return ""
	}

	m, err = multihash.Cast(c)
	if err == nil {
		return m.B58String()
	}

	return ""
}

// ToPeerAddr returns b32-encoded ID. it converts to b32 if B58-encoded.
func ToPeerAddr(s string) string {
	m, err := multihash.FromB58String(s)
	if err == nil {
		return strings.ToLower(crock32.Encode(m))
	}

	//normalize/validate
	d, err := crock32.Decode(s)
	if err == nil {
		m, err = multihash.Cast(d)
		if err != nil {
			return ""
		}
		return strings.ToLower(crock32.Encode(m))
	}

	return ""
}

// ParseInt parses s into int
func ParseInt(s string, v int) int {
	if s == "" {
		return v
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		i = v
	}
	return i
}

// TLD returns last part of a domain name
func TLD(domain string) string {
	sa := strings.Split(domain, ".")
	s := sa[len(sa)-1]

	return s
}

// PeerTLD splits and returns peer id after strippig off m3
func PeerTLD(domain string) string {
	sa := strings.Split(domain, ".")
	s := sa[len(sa)-1]
	if s == "m3" {
		s = sa[len(sa)-2]
	}
	return s
}