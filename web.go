package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gostones/lib"
	host "github.com/libp2p/go-libp2p-host"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/parnurzeal/gorequest"
)

type Health struct {
	ID        string `json:"id"`
	Addr      string `json:"addr"`
	Healthy   bool   `json:"healthy"`
	Timestamp int64  `json:"timestamp"`
}

func HealthHandlerFunc(h host.Host) http.HandlerFunc {
	const elapse int64 = 60000 //one min
	last := ToTimestamp(time.Now())
	healthy := false
	mutex := &sync.Mutex{}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()

		now := ToTimestamp(time.Now())
		if !healthy || now-last > elapse {
			healthy = true //TODO pingProxy(proxyURL)
			last = now
		}
		pid := h.ID().Pretty()
		m := &Health{
			ID:        pid,
			Addr:      ToPeerAddr(pid),
			Healthy:   healthy,
			Timestamp: now,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		b, _ := json.Marshal(m)
		fmt.Fprintf(w, string(b))
	})
}

func pingProxy(proxy string) bool {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	testSite := []string{
		"https://www.google.com/",
		"https://aws.amazon.com/",
		"https://azure.microsoft.com/",
	}

	request := gorequest.New().Proxy(proxy)

	//
	err := lib.Retry(func() error {
		idx := rnd.Intn(len(testSite))
		resp, _, errs := request.
			Head(testSite[idx]).
			End()

		logger.Printf("Proxy: %v response: %v err %v\n", proxy, resp, errs)
		if len(errs) > 0 {
			return errs[0]
		}
		return nil
	})

	return err == nil
}

const pacFile = `
function FindProxyForURL(url, host) {
	return "PROXY %v";
}
`

// PACHandlerFunc handles PAC file request
func PACHandlerFunc(port int) http.HandlerFunc {
	proxyURL := fmt.Sprintf("http://127.0.0.1:%v", port)
	URL, _ := url.Parse(proxyURL)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")
		s := fmt.Sprintf(pacFile, URL.Host)
		w.Write([]byte(s))
	})
}

type PeerInfo struct {
	ID string `json:"id"`
}

// FingerHandlerFunc shows peer information
func FingerHandlerFunc(h host.Host) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", h.ID().Pretty()))
		addrs := h.Addrs()

		for _, addr := range addrs {
			a := addr.Encapsulate(hostAddr)
			logger.Println(a)
		}
		m := &PeerInfo{
			ID: h.ID().Pretty(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		b, _ := json.Marshal(m)
		fmt.Fprintf(w, string(b))
	})
}

// MuxHandlerFunc multiplexes requests
func MuxHandlerFunc(host host.Host, port int) http.HandlerFunc {
	mux := http.NewServeMux()
	mux.HandleFunc("/proxy.pac", PACHandlerFunc(port))
	mux.HandleFunc("/health", HealthHandlerFunc(host))
	mux.HandleFunc("/finger", FingerHandlerFunc(host))

	fs := http.FileServer(http.Dir("public"))
	mux.Handle("/", fs)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mux.ServeHTTP(w, req)
	})
}
