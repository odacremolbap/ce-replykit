package main

import "encoding/json"

// ReplyInstruction from incoming events.
type ReplyInstruction struct {
	Condition string          `json:"condition,omitempty"`
	Action    string          `json:"action,omitempty"`
	Reply     json.RawMessage `json:"reply,omitempty"`
}

// ReplyInstructions is an ordered set of ReplyInstruction
type ReplyInstructions []ReplyInstruction
