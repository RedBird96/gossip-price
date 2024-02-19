package protocol

import (
	"encoding/binary"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	common2 "gossip-price/core/global"
	"math"
	"time"
)

type Signature []byte

func (s Signature) MarshalJSON() ([]byte, error) {
	return []byte("\"" + common.Bytes2Hex(s) + "\""), nil
}

func (s *Signature) UnmarshalJSON(input []byte) error {
	var hexString string
	err := json.Unmarshal(input, &hexString)
	if err != nil {
		return err
	}
	*s = common.Hex2Bytes(hexString)
	return nil
}

func (s Signature) String() string {
	return common.Bytes2Hex(s)
}

type ReceivedMessage struct {
	From    string
	Author  common.Address
	Topic   string
	Data    []byte
	Message SignedMessage
}

func (m ReceivedMessage) MarshalJSON() ([]byte, error) {
	data := make(map[string]any)
	data["from"] = m.From
	data["author"] = m.Author.String()
	data["topic"] = m.Topic
	data["data"] = string(m.Data)
	return json.Marshal(data)
}

type SignedMessage interface {
	MarshalJSON() ([]byte, error)
	UnmarshalJSON([]byte) error
}

type UnsignedMessage interface {
	Sign(key crypto.PrivKey) (SignedMessage, error)
}

type Transport interface {
	Broadcast(message UnsignedMessage) (SignedMessage, error)
	Message() <-chan ReceivedMessage
}

type ProtocolMessage struct {
	MsgId      string
	Price      float64
	Signer     common.Address
	Signature  Signature
	SignedTime time.Time
}

func (p ProtocolMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"id":          p.MsgId,
		"price":       p.Price,
		"signer":      p.Signer,
		"signature":   p.Signature,
		"signed_time": p.SignedTime,
	})
}

func (p *ProtocolMessage) UnmarshalJSON(data []byte) error {
	var temp struct {
		ID         string         `json:"id"`
		Price      float64        `json:"price"`
		Signer     common.Address `json:"signer"`
		Signature  Signature      `json:"signature"`
		SignedTime string         `json:"signed_time"`
	}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	p.MsgId = temp.ID
	p.Price = temp.Price
	p.Signer = temp.Signer
	p.Signature = temp.Signature
	p.SignedTime, _ = time.Parse(time.RFC3339, temp.SignedTime)
	return nil
}

func (p *ProtocolMessage) Sign(key crypto.PrivKey) (SignedMessage, error) {
	datetime := time.Now()
	priceData := make([]byte, 8)
	binary.BigEndian.PutUint64(priceData, math.Float64bits(p.Price))

	bytes, err := key.Sign(priceData)
	if err != nil {
		return nil, err
	}
	pid, err := peer.IDFromPublicKey(key.GetPublic())
	if err != nil {
		return nil, err
	}
	p.Signer = common2.PeerIDToAddress(pid)
	p.Signature = bytes
	p.SignedTime = datetime
	return p, nil
}
