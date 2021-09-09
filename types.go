package main

type Payload struct {
}

type ReplyInstruction struct {
	Condition string
	Action    string
	Payload   Payload
}

type ReplyInstructions []ReplyInstruction
