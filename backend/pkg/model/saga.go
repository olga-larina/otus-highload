package model

import (
	"encoding/json"
)

type SagaEventType string

const (
	UserCreatedSagaEventType            SagaEventType = "UserCreated"
	UserVerifiedSagaEventType           SagaEventType = "UserVerifiedEvent"
	UserVerificationFailedSagaEventType SagaEventType = "UserVerificationFailedEvent"
)

type SagaEvent struct {
	Type    SagaEventType `json:"saga_event_type"`
	Payload any           `json:"payload"`
}

type SagaEventJson struct {
	Type    SagaEventType   `json:"saga_event_type"`
	Payload json.RawMessage `json:"payload"`
}
