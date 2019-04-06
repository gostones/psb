package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	mrand "math/rand"
	"net"
	"sync"

	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	golog "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-discovery"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	p2pnet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	ma "github.com/multiformats/go-multiaddr"
	gologging "github.com/whyrusleeping/go-logging"
)

// Protocol defines the libp2p for proxy
const Protocol = "/dhnt/0.0.1"
const Rendezvous = "dahenitozu"

// InitHost creates a LibP2P host with a random peer ID listening on the
// given multiaddress. It will use secio if secio is true. It will bootstrap using the
// provided PeerInfo
func InitHost() (*HostInfo, error) {
	golog.SetAllLoggers(gologging.ERROR) // Change to DEBUG for extra info
	port := FreePort()

	const randseed = 0

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
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)),
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
	kdht := dht.NewDHT(ctx, basicHost, dstore)

	// Make the routed host
	routedHost := rhost.Wrap(basicHost, kdht)

	// // connect to the chosen ipfs nodes
	// err = bootstrapConnect(ctx, routedHost, IPFS_PEERS)
	// if err != nil {
	// 	return nil, err
	// }

	// Bootstrap the host
	err = kdht.Bootstrap(ctx)
	if err != nil {
		return nil, err
	}

	//
	bootstrapPeer(routedHost, ctx)

	// //
	logger.Info("Announcing ourselves...")
	disco := discovery.NewRoutingDiscovery(kdht)
	discovery.Advertise(ctx, disco, Rendezvous)
	logger.Debug("Successfully announced!")

	//
	Every(15).Seconds().Run(func() {
		logger.Debug("discovering peer ....")

		discover(routedHost, ctx, disco)
	})

	// Build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", routedHost.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	// addr := routedHost.Addrs()[0]
	addrs := routedHost.Addrs()
	logger.Println("I can be reached at:")
	for _, addr := range addrs {
		logger.Println(addr.Encapsulate(hostAddr))
	}

	// log.Printf("Now run \"./routed-echo -l %d -d %s%s\" on a different terminal\n", listenPort+1, routedHost.ID().Pretty(), globalFlag)
	logger.Printf("Initialized. port: %d id: %s", port, routedHost.ID().Pretty())

	return &HostInfo{Port: port, Host: routedHost, DHT: kdht}, nil
}

// func initHost() (*HostInfo, error) {
// 	// LibP2P code uses golog to log messages. They log with different
// 	// string IDs (i.e. "swarm"). We can control the verbosity level for
// 	// all loggers with:
// 	golog.SetAllLoggers(gologging.INFO) // Change to DEBUG for extra info

// 	// // Parse options from the command line
// 	// listenF := flag.Int("l", 0, "wait for incoming connections")
// 	// target := flag.String("d", "", "target peer to dial")
// 	// seed := flag.Int64("seed", 0, "set random seed for id generation")
// 	// global := flag.Bool("global", false, "use global ipfs peers for bootstrapping")
// 	// flag.Parse()

// 	// if *listenF == 0 {
// 	// 	log.Fatal("Please provide a port to bind on with -l")
// 	// }

// 	// const global = true
// 	port := FreePort()

// 	// // Make a host that listens on the given multiaddress
// 	// var bootstrapPeers []pstore.PeerInfo
// 	// // var globalFlag string
// 	// if global {
// 	// 	logger.Println("using global bootstrap")
// 	// 	// bootstrapPeers = IPFS_PEERS
// 	// 	bootstrapPeers = dht.DefaultBootstrapPeers
// 	// 	// globalFlag = " -global"
// 	// } else {
// 	// 	logger.Println("using local bootstrap")
// 	// 	bootstrapPeers = getLocalPeerInfo()
// 	// 	// globalFlag = ""
// 	// }

// 	ha, err := makeRoutedHost(port)
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }
// 	return ha, err
// }

func servePeer(ha host.Host, target string) {
	// Set a stream handler on host A. with
	// protocol name.
	handler := func(s p2pnet.Stream) {
		logger.Printf("Got new request: %v", s.Conn().RemotePeer().Pretty())

		client, err := net.Dial("tcp", target)
		if err != nil {
			logger.Printf("failed to dial %v: %v", target, err)
			return
		}

		logger.Printf("connected: %v -> %v", s, target)

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
		logger.Debugf("failed to decode %v", target)
		return nil, err
	}
	logger.Infof("dialing %v ...", peerid.Pretty())

	// peerinfo := pstore.PeerInfo{ID: peerid}
	// log.Println("opening stream")
	// make a new stream from host B to host A
	// it should be handled on host A by the handler we set above because
	// we use the same protocol
	s, err := ha.NewStream(context.Background(), peerid, Protocol)
	if err != nil {
		logger.Debugf("failed to create stream to %v: %v", peerid.Pretty(), err)
		return nil, err
	}

	return PeerConn{s}, nil
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

func bootstrapPeer(ha host.Host, ctx context.Context) {
	//ctx := context.Background()

	var wg sync.WaitGroup
	for _, peerAddr := range dht.DefaultBootstrapPeers {
		wg.Add(1)
		peerinfo, _ := pstore.InfoFromP2pAddr(peerAddr)

		go func(p pstore.PeerInfo) {
			defer wg.Done()

			ha.Peerstore().AddAddrs(p.ID, p.Addrs, pstore.PermanentAddrTTL)
			if err := ha.Connect(ctx, p); err != nil {
				logger.Warnf("failed to bootstrap with %v: %v", p.ID, err)
			} else {
				logger.Infof("bootstrap connection established: %v %v", ha.ID().Pretty(), p.ID.Pretty())
			}
		}(*peerinfo)
	}
	wg.Wait()
}

func discoverPeer(ha host.Host, kdht *dht.IpfsDHT) []string {
	ctx := context.Background()
	disco := discovery.NewRoutingDiscovery(kdht)

	return discover(ha, ctx, disco)
}

func discover(ha host.Host, ctx context.Context, disco *discovery.RoutingDiscovery) []string {

	// discovery.Advertise(ctx, disco, Rendezvous)
	// logger.Debug("Successfully announced!")

	// Now, look for others who have announced
	// This is like your friend telling you the location to meet you.
	logger.Debug("Searching for peers...")
	pl := []string{}

	peerChan, err := disco.FindPeers(ctx, Rendezvous)
	if err != nil {
		logger.Info(err)
		return pl
	}

	for p := range peerChan {
		if p.ID == ha.ID() {
			continue
		}
		logger.Debugf("Found peer: %v", p.ID.Pretty())

		ha.Peerstore().AddAddrs(p.ID, p.Addrs, pstore.PermanentAddrTTL)

		pl = append(pl, p.ID.Pretty())

		// logger.Debug("Connecting to:", peer)
		// stream, err := ha.NewStream(ctx, peer.ID, protocol.ID(config.ProtocolID))

		// if err != nil {
		// 	logger.Warning("Connection failed:", err)
		// 	continue
		// } else {
		// 	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

		// 	go writeData(rw)
		// 	go readData(rw)
		// }

		// logger.Info("Connected to:", peer)
	}

	return pl
}

// 	ctx := context.Background()

// 	// libp2p.New constructs a new libp2p Host. Other options can be added
// 	// here.
// 	// host, err := libp2p.New(ctx,
// 	// 	libp2p.ListenAddrs([]multiaddr.Multiaddr(config.ListenAddresses)...),
// 	// )
// 	// if err != nil {
// 	// 	panic(err)
// 	// }
// 	// logger.Info("Host created. We are:", host.ID())
// 	// logger.Info(host.Addrs())

// 	// Set a function as stream handler. This function is called when a peer
// 	// initiates a connection and starts a stream with this peer.
// 	// host.SetStreamHandler(protocol.ID(config.ProtocolID), handleStream)

// 	// Start a DHT, for use in peer discovery. We can't just make a new DHT
// 	// client because we want each peer to maintain its own local copy of the
// 	// DHT, so that the bootstrapping node of the DHT can go down without
// 	// inhibiting future peer discovery.
// 	// kademliaDHT, err := libp2pdht.New(ctx, host)
// 	// if err != nil {
// 	// 	panic(err)
// 	// }

// 	// // Bootstrap the DHT. In the default configuration, this spawns a Background
// 	// // thread that will refresh the peer table every five minutes.
// 	// logger.Debug("Bootstrapping the DHT")
// 	// if err = kademliaDHT.Bootstrap(ctx); err != nil {
// 	// 	panic(err)
// 	// }

// 	// Let's connect to the bootstrap nodes first. They will tell us about the
// 	// other nodes in the network.
// 	// var wg sync.WaitGroup
// 	// for _, peerAddr := range dht.DefaultBootstrapPeers {
// 	// 	wg.Add(1)
// 	// 	peerinfo, _ := pstore.InfoFromP2pAddr(peerAddr)

// 	// 	go func(p pstore.PeerInfo) {
// 	// 		defer wg.Done()
// 	// 		defer logger.Println(ctx, "bootstrap", ha.ID(), p.ID)

// 	// 		ha.Peerstore().AddAddrs(p.ID, p.Addrs, pstore.PermanentAddrTTL)
// 	// 		if err := ha.Connect(ctx, p); err != nil {
// 	// 			logger.Warnf("failed to bootstrap with %v: %s", p.ID, err)
// 	// 		} else {
// 	// 			logger.Info("Connection established with bootstrap node:", p)
// 	// 		}
// 	// 	}(*peerinfo)
// 	// }
// 	// wg.Wait()

// 	// We use a rendezvous point "meet me here" to announce our location.
// 	// This is like telling your friends to meet you at the Eiffel Tower.
// 	// logger.Info("Announcing ourselves...")
// 	// routingDiscovery := discovery.NewRoutingDiscovery(kdht)
// 	// // discovery.Advertise(ctx, routingDiscovery, Rendezvous)
// 	// // logger.Debug("Successfully announced!")

// 	// // Now, look for others who have announced
// 	// // This is like your friend telling you the location to meet you.
// 	// logger.Debug("Searching for other peers...")
// 	// peerChan, err := routingDiscovery.FindPeers(ctx, Rendezvous)
// 	// if err != nil {
// 	// 	panic(err)
// 	// }

// 	// for peer := range peerChan {
// 	// 	if peer.ID == ha.ID() {
// 	// 		continue
// 	// 	}
// 	// 	logger.Debug("Found peer:", peer)

// 	// 	// logger.Debug("Connecting to:", peer)
// 	// 	// stream, err := ha.NewStream(ctx, peer.ID, protocol.ID(config.ProtocolID))

// 	// 	// if err != nil {
// 	// 	// 	logger.Warning("Connection failed:", err)
// 	// 	// 	continue
// 	// 	// } else {
// 	// 	// 	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

// 	// 	// 	go writeData(rw)
// 	// 	// 	go readData(rw)
// 	// 	// }

// 	// 	// logger.Info("Connected to:", peer)
// 	// }

// }

// addAddrToPeerstore parses a peer multiaddress and adds
// it to the given host's peerstore, so it knows how to
// contact it. It returns the peer ID of the remote peer.
func addAddrToPeerstore(h host.Host, addr string) peer.ID {
	// The following code extracts target's the peer ID from the
	// given multiaddress
	ipfsaddr, err := ma.NewMultiaddr(addr)
	if err != nil {
		logger.Debug(err)
		return ""
	}
	pid, err := ipfsaddr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		logger.Debug(err)
		return ""
	}

	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		logger.Debug(err)
		return ""
	}

	// Decapsulate the /ipfs/<peerID> part from the target
	// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
	targetPeerAddr, _ := ma.NewMultiaddr(
		fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
	targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)

	// We have a peer ID and a targetAddr so we add
	// it to the peerstore so LibP2P knows how to contact it
	h.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)

	return peerid
}
