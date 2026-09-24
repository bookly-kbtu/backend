package domain

import "slices"

type AssistantRequestStatus string

const (
	AssistantReceived AssistantRequestStatus = "received"
	AssistantParsed   AssistantRequestStatus = "parsed"
	AssistantInvalid  AssistantRequestStatus = "invalid"
	AssistantSearched AssistantRequestStatus = "searched"
	AssistantSelected AssistantRequestStatus = "selected"
	AssistantBooked   AssistantRequestStatus = "booked"
	AssistantFailed   AssistantRequestStatus = "failed"
)

var assistantRequestStatuses = []AssistantRequestStatus{
	AssistantReceived, AssistantParsed, AssistantInvalid, AssistantSearched,
	AssistantSelected, AssistantBooked, AssistantFailed,
}

func ParseAssistantRequestStatus(s string) (AssistantRequestStatus, error) {
	return parseEnum("assistant request status", s, assistantRequestStatuses)
}

func (s AssistantRequestStatus) Valid() bool { return slices.Contains(assistantRequestStatuses, s) }
