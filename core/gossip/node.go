package protocol

import (
	"context"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/multiformats/go-multiaddr"
	"gossip-price/core/global"
	"log"
	"sync"
)

type Validator func(ctx context.Context, topic string, id peer.ID, msg *pubsub.Message) pubsub.ValidationResult

// ValidatorSet stores multiple instances of validators that implements
// the pubsub.ValidatorEx functions. Validators are groped by topic.
type ValidatorSet struct {
	validators []Validator
}

type NodeConfig struct {
	Options []libp2p.Option
	Title   string
	NodeKey crypto.PrivKey
}

// Node is a single node in the P2P network. It wraps the libp2p library to
// provide an easier to use and use-case agnostic interface for the pubsub
// system.
type Node struct {
	ctx    context.Context
	id     peer.ID
	mu     sync.Mutex
	waitCh chan error

	host          host.Host
	pubSub        *pubsub.PubSub
	subs          map[string]*Subscription
	disablePubSub bool
	closed        bool
	peerStore     peerstore.Peerstore
	validatorSet  *ValidatorSet

	hostOpts []libp2p.Option
	//pubsubOpts []pubsub.Option
}

func NewNode(config NodeConfig) (*Node, error) {
	ps, err := pstoremem.NewPeerstore()
	if err != nil {
		return nil, fmt.Errorf("libp2p node error, unable to initialize peerstore: %w", err)
	}
	pid, err := peer.IDFromPublicKey(config.NodeKey.GetPublic())
	if err != nil {
		return nil, err
	}
	err = ps.AddPrivKey(pid, config.NodeKey)
	if err != nil {
		return nil, err
	}

	n := &Node{
		id:        pid,
		waitCh:    make(chan error),
		peerStore: ps,
		subs:      make(map[string]*Subscription),
		closed:    false,
		hostOpts:  config.Options,
	}
	return n, nil
}

// Start the node for connecting and listening from other one
func (n *Node) Start(ctx context.Context) error {
	if n.ctx != nil {
		return errors.New("service can be started only once")
	}
	if ctx == nil {
		return errors.New("context must not be nil")
	}
	n.ctx = ctx

	go n.contextCancelHandler()

	var err error
	n.host, err = libp2p.New(n.hostOpts...)
	if err != nil {
		return fmt.Errorf("libp2p node error, unable to initialize libp2p: %w", err)
	}

	for _, addr := range n.ConnecetedAddressStrings() {
		log.Printf("Node url: %s", addr)
	}
	log.Printf("Node address: %s", global.PeerIDToAddress(n.id))

	if !n.disablePubSub {
		options := []pubsub.Option{
			pubsub.WithMessageAuthor(n.id),
		}
		n.pubSub, err = pubsub.NewGossipSub(n.ctx, n.host, options...)
		if err != nil {
			return fmt.Errorf("libp2p node error, unable to initialize gosspib pubsub: %w", err)
		}
	}

	return nil
}

// Wait waits until the context is canceled or until an error occurs.
func (n *Node) Wait() <-chan error {
	return n.waitCh
}

func (n *Node) Addrs() []multiaddr.Multiaddr {
	var addrs []multiaddr.Multiaddr
	for _, s := range n.ConnecetedAddressStrings() {
		addrs = append(addrs, multiaddr.StringCast(s))
	}
	return addrs
}

func (n *Node) Host() host.Host {
	return n.host
}

func (n *Node) PubSub() *pubsub.PubSub {
	return n.pubSub
}

func (n *Node) Peerstore() peerstore.Peerstore {
	return n.peerStore
}

// Connect to another node using P2P library
func (n *Node) Connect(maddr multiaddr.Multiaddr) error {
	pi, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return err
	}
	err = n.host.Connect(n.ctx, *pi)
	if err != nil {
		return err
	}
	return nil
}

// Subscribe to topic channel and validate
func (n *Node) Subscribe(topic string) (*Subscription, error) {
	if n.pubSub == nil {
		return nil, global.ErrPubSubDisabled
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.closed {
		return nil, fmt.Errorf("libp2p node error: %w", global.ErrConnectionClosed)
	}
	if _, ok := n.subs[topic]; ok {
		return nil, fmt.Errorf("libp2p node error: %w", global.ErrAlreadySubscribed)
	}

	sub, err := newSubscription(n, topic, n.validator(topic))
	if err != nil {
		return nil, err
	}
	n.subs[topic] = sub

	return sub, nil
}

func (n *Node) Unsubscribe(topic string) error {
	if n.pubSub == nil {
		return global.ErrPubSubDisabled
	}
	if n.closed {
		return fmt.Errorf("libp2p node error: %w", global.ErrConnectionClosed)
	}

	sub, err := n.Subscription(topic)
	if err != nil {
		return err
	}

	return sub.close()
}

func (n *Node) Subscription(topic string) (*Subscription, error) {
	if n.pubSub == nil {
		return nil, global.ErrPubSubDisabled
	}
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.closed {
		return nil, fmt.Errorf("libp2p node error: %w", global.ErrConnectionClosed)
	}
	if sub, ok := n.subs[topic]; ok {
		return sub, nil
	}

	return nil, fmt.Errorf("libp2p node error: %w", global.ErrNotSubscribed)
}

// contextCancelHandler handles context cancellation.
func (n *Node) contextCancelHandler() {
	defer func() { close(n.waitCh) }()
	defer log.Print("Stopped")
	<-n.ctx.Done()

	n.mu.Lock()
	defer n.mu.Unlock()

	n.subs = nil
	n.closed = true
	err := n.host.Close()
	if err != nil {
		n.waitCh <- err
	}
}

func (n *Node) AddValidator(validator Validator) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.validatorSet.Add(validator)
}

func (n *Node) Discovery(maddrs []core.Multiaddr) error {
	var err error
	var kadDHT *dht.IpfsDHT

	addrs, err := peer.AddrInfosFromP2pAddrs(maddrs...)
	if err != nil {
		return err
	}

	for _, addr := range addrs {
		// Bootstrap nodes are not protected by KAD-DHT, so we have
		// done it manually.
		//n.connmgr.Protect(addr.ID, "bootstrap")
		n.peerStore.AddAddrs(addr.ID, addr.Addrs, peerstore.PermanentAddrTTL)
	}
	kadDHT, err = dht.New(n.ctx, n.host, dht.BootstrapPeers(addrs...), dht.Mode(dht.ModeServer))
	if err != nil {
		return nil
	}
	if err = kadDHT.Bootstrap(n.ctx); err != nil {
		return nil
	}
	//n.pubsubOpts = append(n.pubsubOpts, pubsub.WithDiscovery(routing.NewRoutingDiscovery(kadDHT)))
	return nil
}

// ConnecetedAddressStrings returns all node's listen multiaddresses as a string list.
func (n *Node) ConnecetedAddressStrings() []string {
	var strs []string
	for _, addr := range n.host.Addrs() {
		strs = append(strs, fmt.Sprintf("%s/p2p/%s", addr.String(), n.host.ID()))
	}
	return strs
}

// validator validates message of specific topic, and returns true or false
func (n *Node) validator(topic string) pubsub.Validator {
	return func(ctx context.Context, id peer.ID, psMsg *pubsub.Message) bool {
		// Validator unmarshalls messages, and unmarshalled message is stored in ValidatorData field
		// which will be used when receives messages
		msg := &ProtocolMessage{}
		err := msg.UnmarshalJSON(psMsg.Data)
		if err != nil {
			return false
		}
		psMsg.ValidatorData = msg
		return true
	}
}

// Add adds new pubsub.ValidatorEx to the set.
func (n *ValidatorSet) Add(validator ...Validator) {
	n.validators = append(n.validators, validator...)
}

// Validator returns function that implements pubsub.ValidatorEx. That function
// will invoke all registered validators for given topic.
func (n *ValidatorSet) Validator(title string) pubsub.ValidatorEx {
	return func(ctx context.Context, id peer.ID, psMsg *pubsub.Message) pubsub.ValidationResult {
		for _, validator := range n.validators {
			if result := validator(ctx, title, id, psMsg); result != pubsub.ValidationAccept {
				return result
			}
		}
		return pubsub.ValidationAccept
	}
}
