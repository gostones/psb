package main

import (
	"flag"
)

func main() {
	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	// golog.SetAllLoggers(gologging.INFO) // Change to DEBUG for extra info

	// Parse options from the command line
	port := flag.Int("port", 8080, "port listening for incoming connections")
	forward := flag.String("forward", "localhost:80", "service host:port to forward to")

	// target := flag.String("d", "", "target peer to dial")
	// seed := flag.Int64("seed", 0, "set random seed for id generation")
	// global := flag.Bool("global", false, "use global ipfs peers for bootstrapping")
	flag.Parse()

	StartProxy(*port, *forward)

	// if *listenF == 0 {
	// 	logger.Fatal("Please provide a port to bind on with -l")
	// }

	// // Make a host that listens on the given multiaddress
	// var bootstrapPeers []pstore.PeerInfo
	// var globalFlag string
	// if *global {
	// 	log.Println("using global bootstrap")
	// 	bootstrapPeers = IPFS_PEERS
	// 	globalFlag = " -global"
	// } else {
	// 	log.Println("using local bootstrap")
	// 	bootstrapPeers = getLocalPeerInfo()
	// 	globalFlag = ""
	// }
	// ha, err := makeRoutedHost(*listenF, *seed, bootstrapPeers, globalFlag)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // Set a stream handler on host A. with
	// // protocol name.
	// ha.SetStreamHandler(Protocol, func(s net.Stream) {
	// 	log.Println("Got a new stream!")
	// 	if err := doEcho(s); err != nil {
	// 		log.Println(err)
	// 		s.Reset()
	// 	} else {
	// 		s.Close()
	// 	}
	// })

	// if *target == "" {
	// 	log.Println("listening for connections")
	// 	select {} // hang forever
	// }
	/**** This is where the listener code ends ****/

	// peerid, err := peer.IDB58Decode(*target)
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// // peerinfo := pstore.PeerInfo{ID: peerid}
	// log.Println("opening stream")
	// // make a new stream from host B to host A
	// // it should be handled on host A by the handler we set above because
	// // we use the same protocol
	// s, err := ha.NewStream(context.Background(), peerid, Protocol)

	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// _, err = s.Write([]byte("Hello, world!\n"))
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// out, err := ioutil.ReadAll(s)
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// log.Printf("read reply: %q\n", out)
}

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
