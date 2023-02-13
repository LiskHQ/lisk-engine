package p2p

import (
	"context"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// MessageValidator can be used to register a new validator for the GossipSub.
type MessageValidator struct {
	validator pubsub.ValidatorEx
	opts      []pubsub.ValidatorOpt
}

// Validator is an interface to be implemented for each type.
type Validator interface {
	ValidateMessage(context.Context, *Message) pubsub.ValidationResult
}

// NewValidator returns a MessageValidator using `mv` for message validation.
func NewValidator(mv Validator, opts ...pubsub.ValidatorOpt) *MessageValidator {
	return &MessageValidator{
		opts: opts,
		validator: func(ctx context.Context, p peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
			lskMsg := new(Message)
			err := lskMsg.Decode(msg.GetData())
			if err != nil {
				return pubsub.ValidationReject
			}

			return mv.ValidateMessage(ctx, lskMsg)
		},
	}
}

func (mtv *MessageValidator) Validator() pubsub.ValidatorEx {
	return mtv.validator
}

func (mtv *MessageValidator) Opts() []pubsub.ValidatorOpt {
	return mtv.opts
}
