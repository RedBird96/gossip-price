package protocol

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	"time"
)

type Options func(n *Node) error

// DialTimeout sets dial timeout for libp2p nodes.
func DialTimeout(t time.Duration) Options {
	return func(n *Node) error {
		n.hostOpts = append(n.hostOpts, libp2p.WithDialTimeout(t))
		return nil
	}
}

// ListenAddrs configures node to listen on the given addresses.
func ListenAddrs(addrs []multiaddr.Multiaddr) Options {
	return func(n *Node) error {
		n.hostOpts = append(n.hostOpts, libp2p.ListenAddrs(addrs...))
		return nil
	}
}

// ExternalAddr configures node to advertise the given addresses.
func ExternalAddr(addr multiaddr.Multiaddr) Options {
	return func(n *Node) error {
		if addr != nil {
			addressFactory := func(addrs []multiaddr.Multiaddr) []multiaddr.Multiaddr { return append(addrs, addr) }
			n.hostOpts = append(n.hostOpts, libp2p.AddrsFactory(addressFactory))
		}
		return nil
	}
}

// PeerPrivKey configures node to use given key as its identity.
func PeerPrivKey(sk crypto.PrivKey) Options {
	return func(n *Node) error {
		n.hostOpts = append(n.hostOpts, libp2p.Identity(sk))
		return nil
	}
}
