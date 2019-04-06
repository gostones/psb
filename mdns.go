package main

import (
	"context"
	"time"

	host "github.com/libp2p/go-libp2p-host"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p/p2p/discovery"
)

type discoveryNotifee struct {
	PeerChan chan pstore.PeerInfo
}

//interface to be called when new  peer is found
func (n *discoveryNotifee) HandlePeerFound(pi pstore.PeerInfo) {
	n.PeerChan <- pi
}

//Initialize the MDNS service
func initMDNS(ctx context.Context, peerhost host.Host, rendezvous string) chan pstore.PeerInfo {
	// An hour might be a long long period in practical applications. But this is fine for us
	s, err := discovery.NewMdnsService(ctx, peerhost, time.Minute, rendezvous)
	if err != nil {
		// panic(err)
		return nil
	}

	//register with service so that we get notified about peer discovery
	n := &discoveryNotifee{}
	n.PeerChan = make(chan pstore.PeerInfo)

	s.RegisterNotifee(n)
	return n.PeerChan
}

func discoverLocal(ctx context.Context, ho host.Host) []string {

	peerChan := initMDNS(ctx, ho, Rendezvous)
	pl := []string{}

	if peerChan == nil {
		return pl
	}

	logger.Debugln("mdns searching for peers ...")

	for p := range peerChan {
		if p.ID == ho.ID() {
			continue
		}
		logger.Debugf("mdns found peer: %v", p.ID.Pretty())

		ho.Peerstore().AddAddrs(p.ID, p.Addrs, pstore.PermanentAddrTTL)

		pl = append(pl, p.ID.Pretty())
	}

	return pl
}
