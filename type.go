package main

import (
	p2pnet "github.com/libp2p/go-libp2p-net"
	"net"
)

type PeerAddr struct {
	id string
}

func (r PeerAddr) Network() string {
	return "ipfs"
}

func (r PeerAddr) String() string {
	return r.id
}

type PeerConn struct {
	p2pnet.Stream
}

func (r PeerConn) LocalAddr() net.Addr {
	return PeerAddr{id: string(r.Conn().LocalPeer())}
}

func (r PeerConn) RemoteAddr() net.Addr {
	return PeerAddr{id: string(r.Conn().RemotePeer())}
}
