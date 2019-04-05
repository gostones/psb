// https://github.com/elazarl/goproxy
package main

import (
	"bytes"
	"fmt"
	"github.com/elazarl/goproxy"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

func redirectHost(r *http.Request, host, body string) *http.Response {
	resp := &http.Response{}
	resp.Request = r
	resp.TransferEncoding = r.TransferEncoding
	resp.Header = make(http.Header)
	resp.Header.Add("Content-Type", "text/plain")

	u := *r.URL
	u.Host = host
	resp.Header.Set("Location", u.String())

	resp.StatusCode = http.StatusMovedPermanently
	resp.Status = http.StatusText(resp.StatusCode)
	buf := bytes.NewBufferString(body)
	resp.ContentLength = int64(buf.Len())
	resp.Body = ioutil.NopCloser(buf)
	return resp
}

// func cors(r *http.Response) {
// 	r.Header.Set("Access-Control-Allow-Origin", "*")
// 	r.Header.Set("Access-Control-Allow-Credentials", "true")
// 	r.Header.Set("Access-Control-Allow-Methods", "*")
// 	r.Header.Set("Access-Control-Allow-Headers", "*")
// }

// StartProxy dispatches request based on network addr
func StartProxy(port int, local string) {
	ha, err := initHost()
	if err != nil {
		logger.Fatal(err)
	}

	servePeer(ha, local)

	//
	proxy := goproxy.NewProxyHttpServer()
	dial := func(network, addr string) (net.Conn, error) {
		hostport := strings.Split(addr, ":")

		// // resolved := hostport[0] //nb.ResolveAddr(hostport[0])
		// be, viaProxy := route.Match(hostport[0])
		// if be == nil || len(be) == 0 {
		// 	return nil, fmt.Errorf("Proxy routing error: %v %v", network, addr)
		// }
		// logger.Debugf("Router.Match(%q): %v proxy: %v, network: %v addr: %v", hostport[0], *be[0], viaProxy, network, addr)

		// // prevent loop
		// if be[0].Hostname == hostport[0] {
		// 	return net.Dial(network, addr)
		// }

		// if be[0].Hostname == "direct" {
		// 	return net.Dial(network, addr)
		// }

		// if be[0].Hostname == "peer" {
		// 	// logger.Debugf("@@@ Dial peer network: %v addr: %v\n", network, addr)
		tld := PeerTLD(hostport[0])
		pid := ToPeerID(tld)
		if pid == "" {
			return nil, fmt.Errorf("Peer invalid: %v", hostport[0])
		}
		logger.Debugf("@@@ Dial peer network: %v addr: %v pid: %v\n", network, addr, pid)

		return dialPeer(ha, pid)
		
		// 	// target := nb.GetPeerTarget(id)
		// 	// if target == "" {
		// 	// 	return nil, fmt.Errorf("Peer not reachable: %v", hostport[0])
		// 	// }

		// logger.Debugf("@@@ Dial peer network: %v addr: %v pid: %v\n", network, addr, pid)
		// dial := proxy.NewConnectDialToProxy(fmt.Sprintf("http://%v", target))

		

		// 	// if dial != nil {
		// 	// 	return dial(network, addr)
		// 	// }
		// 	return nil, fmt.Errorf("Peer proxy error: %v", hostport)
		// }

		// // pass on port if not provided in backend target
		// port := fmt.Sprintf("%v", be[0].Port)
		// if be[0].Port == 0 {
		// 	port = hostport[1]
		// }
		// target := fmt.Sprintf("%v:%v", be[0].Hostname, port)
		// if viaProxy {
		// 	dial := proxy.NewConnectDialToProxy(fmt.Sprintf("http://%v", target))

		// 	if dial != nil {
		// 		return dial(network, addr)
		// 	}
		// 	return nil, fmt.Errorf("Proxy routing error: %v %v", network, addr)
		// }

		// return net.Dial(network, target)
	}

	//
	proxy.ConnectDial = nil
	proxy.Tr.Dial = dial
	proxy.Tr.DialTLS = nil
	proxy.Tr.Proxy = nil
	proxy.NonproxyHandler = MuxHandlerFunc(fmt.Sprintf("http://127.0.0.1:%v", port))

	//
	proxy.Verbose = true

	// auth
	// auth.ProxyBasic(proxy, "m3_realm", func(user, passwd string) bool {
	// 	//TODO verify peer is who it claims to be
	// 	//user is the peer id and pwd is: peer_addr,timestamp
	// 	//after decrypting with peer's public key
	// 	//return user == json[0]
	// 	return true
	// })

	proxy.OnRequest().DoFunc(
		func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			logger.Debugf("\n\n\n##################\n")
			logger.Debugf("@@@ OnRequest Proto: %v method: %v url: %v\n", req.Proto, req.Method, req.URL)
			logger.Debugf("@@@ OnRequest request: %v\n", req)

			return req, nil
		})

	proxy.OnResponse().DoFunc(func(r *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		logger.Debugf("\n--------------------\n")
		// if r != nil {
		// 	r.Header.Add("X-Peer-Id", nb.My.ID)
		// 	cors(r)
		// 	logger.Debugf("@@@ Proxy OnResponse status: %v length: %v\n", r.StatusCode, r.ContentLength)
		// }
		logger.Debugf("@@@ OnResponse response: %v\n", r)
		return r
	})

	logger.Debugf("Proxy listening on: %v\n", port)
	logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), proxy))
}

// // StartProxy starts proxy services
// func StartProxy() {
// 	// logger.Infof("Configuration: %v", cfg)

// 	// route := NewRouteRegistry()
// 	// route.ReadFile(cfg.RouteFile)

// 	// //
// 	// port := cfg.Port
// 	// logger.Infof("proxy port: %v\n", port)

// 	// HTTPProxy(port, route)
// }
