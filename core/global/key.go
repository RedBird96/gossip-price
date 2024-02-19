package global

import (
	common2 "github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
)

func PeerIDToAddress(id peer.ID) common2.Address {
	return common2.BytesToAddress([]byte(id))
}
