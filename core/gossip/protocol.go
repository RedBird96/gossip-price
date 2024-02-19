// This file is part of the core logic for building P2P protocol
// You should create p2p protocol
package protocol

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/multiformats/go-multiaddr"
	"gossip-price/core/global"
	"strings"
	"time"
)

const ProtocolName = "gossipProtocol"

// defaultListenAddrs is the list of default addresses on which node will
// be listening on.
var defaultListenAddrs = []string{"/ip4/0.0.0.0/tcp/0"}

const timeout = 1000 * time.Second

// Values for the connection limiter:
const minConnections = 100
const maxConnections = 150

// Config is the configuration for the Protocol communication.
type Config struct {
	// ConnectedAddress is a list of multiaddresses on which this node will be
	// listening on. If empty, the localhost, and a random port will be used.
	ConnectedAddress []string
	// BootstrapAddress is a list multiaddresses of initial peers to connect to.
	// This option is ignored when discovery is disabled.
	BootstrapAddress []string
	// NodeKey is a key used for peer identity and sign message. If empty, then random key
	// is used. Ignored in bootstrap mode.
	NodeKey crypto.PrivKey
	// Titles is a list of subscribed topics. A value of the map a type of
	// message given as a nil pointer, e.g.: (*Message)(nil).
	Titles string
	// IsBootstrap is a flag used for identify if it's bootstrap or not
	IsBootstrap bool
}

// Protocol is the wrapper for the Node that implements the communication.
type Protocol struct {
	id          peer.ID
	node        *Node
	titles      string
	isBootstrap bool
	bootAddress []string
	msgCh       chan ReceivedMessage
}

// New returns a new instance of a transport, implemented with
// the protocol library.
func New(c Config) (*Protocol, error) {
	var err error
	if c.NodeKey == nil {
		seed := rand.Reader
		if c.IsBootstrap {
			seed = strings.NewReader("QmdErMiygrmkPsLTxzLTNEq5p4kCXSx26encEBkdoYRsGJ")
		}
		c.NodeKey, _, err = crypto.GenerateEd25519Key(seed)
		if err != nil {
			return nil, fmt.Errorf("P2P protocol error, unable to generate a random private key: %w", err)
		}
	}

	listenAddrs, err := convertAddress(c.ConnectedAddress)
	if err != nil {
		return nil, fmt.Errorf("P2P protocol error, unable to parse listenAddrs: %w", err)
	}

	mgr, _ := connmgr.NewConnManager(minConnections, maxConnections, connmgr.WithGracePeriod(5*time.Minute))

	op := []libp2p.Option{
		libp2p.WithDialTimeout(timeout),
		libp2p.ListenAddrs(listenAddrs...),
		libp2p.ConnectionManager(mgr),
		libp2p.Identity(c.NodeKey),
	}

	n, err := NewNode(NodeConfig{
		Options: op,
		NodeKey: c.NodeKey,
		Title:   c.Titles,
	})
	if err != nil {
		return nil, fmt.Errorf("Protocol error, unable to initialize node: %w", err)
	}

	id, err := peer.IDFromPrivateKey(c.NodeKey)
	if err != nil {
		return nil, fmt.Errorf("P2P transport error, unable to get public ID from private key: %w", err)
	}

	return &Protocol{
		id:          id,
		node:        n,
		isBootstrap: c.IsBootstrap,
		bootAddress: c.BootstrapAddress,
		titles:      c.Titles,
		msgCh:       make(chan ReceivedMessage),
	}, nil
}

// Start implements the transport.Transport interface.
func (p *Protocol) Start(ctx context.Context) error {
	if err := p.node.Start(ctx); err != nil {
		return fmt.Errorf("Protocol error, unable to start node: %w", err)
	}
	if p.isBootstrap == false {

		if err := p.subscribe(p.titles); err != nil {
			return err
		}

	}
	if err := p.bootstrap(p.bootAddress); err != nil {
		return fmt.Errorf("Protocol error, unable to start node: %w", err)
	}
	return nil
}

// Broadcast implements the transport.Transport interface.
func (p *Protocol) Broadcast(message UnsignedMessage) (SignedMessage, error) {
	if len(p.titles) == 0 {
		return nil, fmt.Errorf("%w", global.ErrEmptyTitle)
	}
	sub, err := p.node.Subscription(p.titles)
	if err != nil {
		return nil, fmt.Errorf("Protocol error, unable to get subscription for %s topic: %w", p.titles, err)
	}
	sign, err := message.Sign(p.node.peerStore.PrivKey(p.id))
	if err != nil {
		return nil, fmt.Errorf("Protocol error, failed to sign: %w", err)
	}
	data, err := sign.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("Protocol error, unable to marshall message: %w", err)
	}
	err = sub.Publish(data)
	return sign, err
}

// Subscribe to title channel
func (p *Protocol) subscribe(title string) error {
	sub, err := p.node.Subscribe(title)
	if err != nil {
		return fmt.Errorf("Protocol error, unable to subscribe to topic %s: %w", title, err)
	}
	go p.messagesLoop(title, sub)
	return nil
}

// Discovery the bootstrap node for connecting
func (p *Protocol) bootstrap(addrs []string) error {
	bootstraps, _ := convertAddress(addrs)
	return p.node.Discovery(bootstraps)
}

func (p *Protocol) messagesLoop(title string, sub *Subscription) {
	for {
		select {
		case nodeMsg, ok := <-sub.Next():
			if !ok {
				continue
			}
			id := nodeMsg.GetFrom()
			if id.String() == p.id.String() {
				continue
			}
			checkRes, ok := nodeMsg.ValidatorData.(SignedMessage)
			if !ok {
				continue
			}
			msg := ReceivedMessage{
				From:    id.String(),
				Topic:   title,
				Data:    nodeMsg.Data,
				Author:  global.PeerIDToAddress(id),
				Message: checkRes,
			}
			p.msgCh <- msg
		}
	}
}

func (p *Protocol) Message() <-chan ReceivedMessage {
	return p.msgCh
}

// Wait implements the transport.Transport interface.
func (p *Protocol) Wait() <-chan error {
	return p.node.Wait()
}

// strsToMaddrs converts multiaddresses given as strings to a
// list of multiaddr.Multiaddr.
func convertAddress(addrs []string) ([]core.Multiaddr, error) {
	var maddrs []core.Multiaddr
	for _, addrstr := range addrs {
		maddr, err := multiaddr.NewMultiaddr(addrstr)
		if err != nil {
			return nil, err
		}
		maddrs = append(maddrs, maddr)
	}
	return maddrs, nil
}
