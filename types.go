package main

import "encoding/json"

type ReplyInstruction struct {
	Condition string          `json:"condition,omitempty"`
	Action    string          `json:"action,omitempty"`
	Reply     json.RawMessage `json:"reply,omitempty"`
}

type ReplyInstructions []ReplyInstruction
