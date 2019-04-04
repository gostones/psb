package main

import (
	// "bufio"
	"context"
	"crypto/rand"
	// "flag"
	"fmt"
	"io"
	// "io/ioutil"
	"net"
	"log"
	mrand "math/rand"

	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	golog "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	p2pnet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	ma "github.com/multiformats/go-multiaddr"
	gologging "github.com/whyrusleeping/go-logging"
)

const Protocol = "/proxy-m3/0.0.1"

// makeRoutedHost creates a LibP2P host with a random peer ID listening on the
// given multiaddress. It will use secio if secio is true. It will bootstrap using the
// provided PeerInfo
func makeRoutedHost(listenPort int, randseed int64, bootstrapPeers []pstore.PeerInfo) (host.Host, error) {

	// If the seed is zero, use real cryptographic randomness. Otherwise, use a
	// deterministic randomness source to make generated keys stay the same
	// across multiple runs
	var r io.Reader
	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}

	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		libp2p.Identity(priv),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.NATPortMap(),
	}

	ctx := context.Background()

	basicHost, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	// Construct a datastore (needed by the DHT). This is just a simple, in-memory thread-safe datastore.
	dstore := dsync.MutexWrap(ds.NewMapDatastore())

	// Make the DHT
	dht := dht.NewDHT(ctx, basicHost, dstore)

	// Make the routed host
	routedHost := rhost.Wrap(basicHost, dht)

	// connect to the chosen ipfs nodes
	err = bootstrapConnect(ctx, routedHost, bootstrapPeers)
	if err != nil {
		return nil, err
	}

	// Bootstrap the host
	err = dht.Bootstrap(ctx)
	if err != nil {
		return nil, err
	}

	// Build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", routedHost.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	// addr := routedHost.Addrs()[0]
	addrs := routedHost.Addrs()
	log.Println("I can be reached at:")
	for _, addr := range addrs {
		log.Println(addr.Encapsulate(hostAddr))
	}

	// log.Printf("Now run \"./routed-echo -l %d -d %s%s\" on a different terminal\n", listenPort+1, routedHost.ID().Pretty(), globalFlag)
	log.Printf("Initialized. port: %d id: %s", listenPort, routedHost.ID().Pretty())

	return routedHost, nil
}

func initHost() (host.Host, error) {
	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	golog.SetAllLoggers(gologging.INFO) // Change to DEBUG for extra info

	// // Parse options from the command line
	// listenF := flag.Int("l", 0, "wait for incoming connections")
	// target := flag.String("d", "", "target peer to dial")
	// seed := flag.Int64("seed", 0, "set random seed for id generation")
	// global := flag.Bool("global", false, "use global ipfs peers for bootstrapping")
	// flag.Parse()

	// if *listenF == 0 {
	// 	log.Fatal("Please provide a port to bind on with -l")
	// }

	const seed = 0
	const global = true

	listen := FreePort()

	// Make a host that listens on the given multiaddress
	var bootstrapPeers []pstore.PeerInfo
	// var globalFlag string
	if global {
		log.Println("using global bootstrap")
		bootstrapPeers = IPFS_PEERS
		// globalFlag = " -global"
	} else {
		log.Println("using local bootstrap")
		bootstrapPeers = getLocalPeerInfo()
		// globalFlag = ""
	}

	ha, err := makeRoutedHost(listen, seed, bootstrapPeers)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	return ha, err
}

func servePeer(ha host.Host, target string) {
	// Set a stream handler on host A. with
	// protocol name.
	handler := func(s p2pnet.Stream) {
		log.Println("Got a new stream: %v", s.Conn().RemotePeer())
		client, err := net.Dial("tcp", target)
		if err != nil {
			log.Printf("failed to dial: %v", err)
			return
		}

		log.Printf("connected: %v -> %v", s, target)
		go func() {
			defer client.Close()
			defer s.Close()
			io.Copy(client, s)
		}()
		go func() {
			defer client.Close()
			defer s.Close()
			io.Copy(s, client)
		}()
		// if err := doEcho(s); err != nil {
		// 	log.Println(err)
		// 	s.Reset()
		// } else {
		// 	s.Close()
		// }
	}

	ha.SetStreamHandler(Protocol, handler)

	// if *target == "" {
	// 	log.Println("listening for connections")
	// 	select {} // hang forever
	// }
	/**** This is where the listener code ends ****/
}

func dialPeer(ha host.Host, target string) (net.Conn, error) {
	peerid, err := peer.IDB58Decode(target)
	if err != nil {
		// log.Fatalln(err)
		return nil, err
	}

	// peerinfo := pstore.PeerInfo{ID: peerid}
	// log.Println("opening stream")
	// make a new stream from host B to host A
	// it should be handled on host A by the handler we set above because
	// we use the same protocol
	s, err := ha.NewStream(context.Background(), peerid, Protocol)
	return PeerConn{s}, err
}

// func connectPeer(conn net.Conn, ha host.Host, target string) error {
// 	peerid, err := peer.IDB58Decode(target)
// 	if err != nil {
// 		// log.Fatalln(err)
// 		return err
// 	}

// 	// peerinfo := pstore.PeerInfo{ID: peerid}
// 	// log.Println("opening stream")
// 	// make a new stream from host B to host A
// 	// it should be handled on host A by the handler we set above because
// 	// we use the same protocol
// 	s, err := ha.NewStream(context.Background(), peerid, Protocol)

// 	if err != nil {
// 		// log.Fatalln(err)
// 		return err
// 	}

// 	log.Printf("connected: %v -> %v", conn, peerid)

// 	go func() {
//         defer s.Close()
//         defer conn.Close()
//         io.Copy(s, conn)
// 	}()
	
//     go func() {
//         defer s.Close()
//         defer conn.Close()
//         io.Copy(conn, s)
//     }()
	
// 	return nil

// 	// _, err = s.Write([]byte("Hello, world!\n"))
// 	// if err != nil {
// 	// 	log.Fatalln(err)
// 	// }

// 	// out, err := ioutil.ReadAll(s)
// 	// if err != nil {
// 	// 	log.Fatalln(err)
// 	// }

// 	// log.Printf("read reply: %q\n", out)
// }

// // doEcho reads a line of data from a stream and writes it back
// func doEcho(s net.Stream) error {
// 	buf := bufio.NewReader(s)
// 	str, err := buf.ReadString('\n')
// 	if err != nil {
// 		return err
// 	}

// 	log.Printf("read: %s\n", str)
// 	_, err = s.Write([]byte(str))
// 	return err
// }
