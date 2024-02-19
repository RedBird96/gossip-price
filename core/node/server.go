package server

import (
	"context"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gossip-price/core/consensus"
	"gossip-price/core/global"
	protocol "gossip-price/core/gossip"
	"log"
	"time"
)

type Server struct {
	ctx       context.Context
	bootStrap bool
	protocol  *protocol.Protocol
	engine    *consensus.Engine
}

func NewGossipServer() (*Server, error) {
	config := protocol.Config{
		IsBootstrap:      global.GPBootstrapMode,
		Titles:           "ethPrice",
		ConnectedAddress: []string{global.GPConnectionAddress},
		BootstrapAddress: []string{global.GPBootstrapAddress},
	}

	en := consensus.NewEngine()
	pro, err := protocol.New(config)
	if err != nil || en == nil {
		return nil, errors.New("New Gossip Server error")
	}

	return &Server{
		bootStrap: global.GPBootstrapMode,
		protocol:  pro,
		engine:    en,
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	if s.bootStrap {
		log.Print("Bootstrap server started")
	} else {
		log.Print("Goosip server started")
	}
	s.ctx = ctx
	err := s.protocol.Start(ctx)
	if err != nil {
		return err
	}
	if !s.bootStrap {
		go s.Broadcast()
		go s.messageLoop()
		s.engine.StartEngine(ctx)
	}
	return nil
}

func (s *Server) Broadcast() {
	for {
		select {
		case <-s.ctx.Done():
			break
		case <-time.After(time.Duration(global.GPFetchPriceInterval) * time.Second):
			price, _ := global.GetETHPrice()
			if price != 0 {
				id := uuid.New().String()
				p, _ := s.protocol.Broadcast(&protocol.ProtocolMessage{
					MsgId: id,
					Price: price,
				})
				if !s.engine.CheckAlreadySigned(p.(*protocol.ProtocolMessage).MsgId, p.(*protocol.ProtocolMessage).Signer) {
					s.engine.Append(*p.(*protocol.ProtocolMessage))
				}
			}

		}
	}
}

func (s *Server) messageLoop() {
	ch := s.protocol.Message()
	for {
		select {
		case <-s.ctx.Done():
			break
		case msg := <-ch:
			priceMsg, ok := msg.Message.(*protocol.ProtocolMessage)
			if !ok {
				continue
			}
			if s.engine.CheckAlreadySigned(priceMsg.MsgId, priceMsg.Signer) {
				continue
			}
			if s.engine.Append(*priceMsg) {
				_, _ = s.protocol.Broadcast(priceMsg)
			}
		}
	}
}

func (s *Server) Wait() <-chan error {
	return s.protocol.Wait()
}
