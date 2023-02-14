package p2p

import (
	"context"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// ValidationResult represents the decision of a validator.
type ValidationResult = pubsub.ValidationResult

const (
	// ValidationAccept is a validation decision that indicates a valid message that should be accepted and
	// delivered to the application and forwarded to the network.
	ValidationAccept = ValidationResult(0)
	// ValidationReject is a validation decision that indicates an invalid message that should not be
	// delivered to the application or forwarded to the application. Furthermore the peer that forwarded
	// the message should be penalized.
	ValidationReject = ValidationResult(1)
	// ValidationIgnore is a validation decision that indicates a message that should be ignored: it will
	// be neither delivered to the application nor forwarded to the network.
	ValidationIgnore = ValidationResult(2)
)

// MessageValidator can be used to register a new validator for the GossipSub.
type MessageValidator = func(context.Context, peer.ID, *pubsub.Message) ValidationResult

// Validator should be implemented for each type that we want to validate it.
type Validator = func(context.Context, *Message) ValidationResult

// NewValidator returns a MessageValidator using `mv` for message validation.
func NewValidator(mv Validator) MessageValidator {
	return func(ctx context.Context, p peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
		lskMsg := new(Message)
		err := lskMsg.Decode(msg.GetData())
		if err != nil {
			return ValidationReject
		}

		return mv(ctx, lskMsg)

	}
}
