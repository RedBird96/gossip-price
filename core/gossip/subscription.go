package protocol

import (
	"context"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
)

type Subscription struct {
	ctx          context.Context
	ctxCancel    context.CancelFunc
	title        *pubsub.Topic
	subscription *pubsub.Subscription
	cancelRelay  pubsub.RelayCancelFunc
	validatorSet *ValidatorSet
	msgCh        chan *pubsub.Message
}

func newSubscription(node *Node, title string, validator pubsub.Validator) (*Subscription, error) {
	var err error
	ctx, ctxCancel := context.WithCancel(node.ctx)
	s := &Subscription{
		ctx:          ctx,
		ctxCancel:    ctxCancel,
		validatorSet: node.validatorSet,
		msgCh:        make(chan *pubsub.Message),
	}
	err = node.pubSub.RegisterTopicValidator(title, validator)
	if err != nil {
		return nil, err
	}
	s.title, err = node.PubSub().Join(title)
	if err != nil {
		return nil, err
	}
	s.cancelRelay, err = s.title.Relay()
	if err != nil {
		return nil, err
	}
	s.subscription, err = s.title.Subscribe()
	if err != nil {
		return nil, err
	}
	go s.messageLoop()
	return s, err
}

func (s *Subscription) Publish(msg []byte) error {
	if msg == nil {
		return errors.New("Subscription Publish error")
	}

	err := s.title.Publish(s.ctx, msg)

	return err
}

func (s *Subscription) messageLoop() {
	defer close(s.msgCh)
	for {
		msg, err := s.subscription.Next(s.ctx)
		if err != nil {
			// The only time when an error may be returned here is
			// when the subscription is canceled.
			return
		}
		s.msgCh <- msg
	}
}

//func (s *Subscription) validator(title string) pubsub.ValidatorEx {
//	return func(ctx context.Context, id peer.ID, psMsg *pubsub.Message) pubsub.ValidationResult {
//		vr := s.validatorSet.Validator(title)(ctx, id, psMsg)
//		return vr
//	}
//}

func (s *Subscription) close() error {
	s.ctxCancel()
	s.subscription.Cancel()
	s.cancelRelay()
	return s.title.Close()
}

func (s *Subscription) Next() chan *pubsub.Message {
	return s.msgCh
}
