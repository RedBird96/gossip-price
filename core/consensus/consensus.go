package consensus

import (
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"gossip-price/core/consensus/db"
	"gossip-price/core/global"
	protocol "gossip-price/core/gossip"
	"strconv"
	"sync"
	"time"
)

type Engine struct {
	ctx           context.Context
	database      *db.Database
	signerStarter map[string]common.Address
	data          map[string][]protocol.ProtocolMessage
	verifiedData  []protocol.ProtocolMessage
	verifiedMutex sync.Mutex
}

// New returns a new consensus engine of protocol with engine data
func NewEngine() *Engine {
	dbTmp := db.NewDatabase()
	return &Engine{
		database:      dbTmp,
		data:          make(map[string][]protocol.ProtocolMessage),
		signerStarter: make(map[string]common.Address),
		verifiedData:  make([]protocol.ProtocolMessage, 0),
	}
}

// Start verify engine with context
func (m *Engine) StartEngine(ctx context.Context) {
	m.ctx = ctx
	go m.VerifyMessage()
}

// Return the count of current signed
func (m *Engine) GetSignedCount(msgId string) int {
	val, ok := m.data[msgId]
	if !ok || len(val) == 0 {
		return 0
	}
	return len(val)
}

// Check current signer already signed or not
// Return true if current signer already signed
// Return false if it's not signed yet
func (m *Engine) CheckAlreadySigned(msgId string, currentSigner common.Address) bool {
	val, ok := m.data[msgId]
	if !ok || len(val) == 0 {
		return false
	}
	// Check if current signer is existed in the list
	for _, d := range val {
		if currentSigner == d.Signer {
			return true
		}
	}
	return false
}

// Append signed message to cache memory and if the
// signed count is bigger than GPMinimumSignerCount
// then register the msgId to verified Data list
func (m *Engine) Append(message protocol.ProtocolMessage) bool {
	_, ok := m.data[message.MsgId]
	if !ok {
		// When it's first signer, allocate array and set signer as first
		m.data[message.MsgId] = make([]protocol.ProtocolMessage, 0)
		m.signerStarter[message.MsgId] = message.Signer
	}

	m.data[message.MsgId] = append(m.data[message.MsgId], message)
	// Check if the signed counts is bigger than minimum count
	if len(m.data[message.MsgId]) >= global.GPMinimumSignerCount {
		// Lock/Unlock verified to make no change itself while verify the message below function
		m.verifiedMutex.Lock()
		m.verifiedData = append(m.verifiedData, message)
		m.verifiedMutex.Unlock()
		return false
	}
	return true
}

// Every 30 seconds it will check the verified list
// and register to Postgre database if it passed 30 seconds
// from the last signed time
func (m *Engine) VerifyMessage() {
	for {
		select {
		case <-m.ctx.Done():
			break
		case <-time.After(time.Second * 30): //todo
			m.verifiedMutex.Lock()
			cuTime := time.Now().Unix()
			// Create temp data to keep remaining data and remove other ones
			remainData := make([]protocol.ProtocolMessage, 0)
			for _, val := range m.verifiedData {
				lastTime := val.SignedTime.Unix()
				// Check the time if it passed 30 seconds from last signed  time
				if cuTime-lastTime < 30 { //todo
					// If it's not passed 30 seconds it will move to remain list
					remainData = append(remainData, val)
				} else {
					// If it's passed 30 seconds it will store to database
					existed := m.database.ExistCheck(val.MsgId)
					if existed {
						continue
					}
					msgData, _ := m.data[val.MsgId]
					signData := map[string]string{
						"first_Signer":     msgData[0].Signer.String(),
						"first_Signature":  msgData[0].Signature.String(),
						"second_Signer":    msgData[1].Signer.String(),
						"second_Signature": msgData[1].Signature.String(),
						"third_Signer":     msgData[2].Signer.String(),
						"third_Signature":  msgData[2].Signature.String(),
					}
					jsonData, err := json.Marshal(signData)
					_, err = m.database.CreateRate(&db.Rate{
						ID:              val.MsgId,
						Price:           strconv.FormatFloat(val.Price, 'f', 2, 64),
						First_Signer:    m.signerStarter[val.MsgId].String(),
						Sign_Data:       string(jsonData),
						LastSigned_Time: val.SignedTime,
						Created_Time:    time.Now(),
					})
					// If writing database is failed then it will move to remain list as well
					if err != nil {
						remainData = append(remainData, val)
					}
				}
			}
			// Replace the remain list to verified data to verify next time and unlock
			m.verifiedData = remainData
			m.verifiedMutex.Unlock()
			break
		}
	}
}
