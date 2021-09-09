package main

import (
	"container/list"
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/protocol"
)

type value struct {
	n       int
	expires time.Time
	le      *list.Element
}

type RequestCountPerID struct {
	count map[string]*value
	rl    *list.List
	ttl   time.Duration

	m sync.RWMutex
}

func NewRequestCountPerID(ctx context.Context, ttl, gc time.Duration) *RequestCountPerID {
	rc := &RequestCountPerID{
		count: make(map[string]*value),
		rl:    list.New(),
		ttl:   ttl,
	}

	go func() {
		for {
			select {
			case <-time.After(gc):
				rc.RemoveStale()
			case <-ctx.Done():
				return
			}
		}
	}()

	return rc
}

// Increase the request count for the ID at the ephemeral
// in-memory storage. Returns the updated count value.
func (c *RequestCountPerID) Increase(id string) int {
	expires := time.Now().Add(c.ttl)
	c.m.Lock()
	defer c.m.Unlock()

	v, ok := c.count[id]
	if ok {
		if v.expires.Before(time.Now()) {
			// reset value for stale keys.
			ok = false
		}
	}

	if !ok {
		el := c.rl.PushFront(id)
		c.count[id] = &value{
			n:       1,
			expires: expires,
			le:      el,
		}
		return 1
	}

	v.n += 1
	v.expires = expires
	c.rl.MoveToFront(v.le)

	return v.n
}

// Get the request count for the ID
func (c *RequestCountPerID) Get(id string) int {
	c.m.RLock()
	defer c.m.RUnlock()

	v, ok := c.count[id]
	if !ok || v.expires.Before(time.Now()) {
		return 0
	}

	return v.n
}

// Delete the request count for the ID
func (c *RequestCountPerID) Delete(id string) {
	c.m.Lock()
	defer c.m.Unlock()

	if v, ok := c.count[id]; ok {
		c.rl.Remove(v.le)
		delete(c.count, id)
	}
}

func (c *RequestCountPerID) RemoveStale() {
	c.m.Lock()
	defer c.m.Unlock()

	for {
		e := c.rl.Back()
		if e == nil {
			return
		}

		id := e.Value.(string)
		if c.count[id].expires.Before(time.Now()) {
			return
		}
		c.rl.Remove(e)
		delete(c.count, id)
	}
}

// StatefulHandler keeps track of requests calls.
type StatefulHandler struct {
	// requestCount aggregates the number of calls per request ID.
	requestCount *RequestCountPerID
}

func NewStatefulHandler(ctx context.Context) *StatefulHandler {
	return &StatefulHandler{
		requestCount: NewRequestCountPerID(ctx, 30*time.Second, 30*time.Second),
	}
}

func (s *StatefulHandler) Handle(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, protocol.Result) {
	ris := &ReplyInstructions{}
	if err := event.DataAs(ris); err != nil {
		log.Printf("Error parsing reply instructions: %v", err)
		return nil, protocol.ResultNACK
	}

	// increase the count for request calls
	s.requestCount.Increase(event.Context.GetID())

	for _, ri := range *ris {

		ok, err := s.evalCondition(&ri, event)
		if err != nil {
			log.Printf("Error evaluating condition: %v", err)
			return nil, protocol.ResultNACK
		}

		if !ok {
			// if the condition is not true continue to process next instruction.
			continue
		}

		// execute condition
		return s.executeAction(&ri, event)
	}

	log.Printf("Returning default empty ACK")
	return nil, protocol.ResultACK
}

func (s *StatefulHandler) evalCondition(ri *ReplyInstruction, event cloudevents.Event) (bool, error) {
	kv := strings.Split(ri.Condition, ":")

	switch kv[0] {
	case "always", "":
		// Absence of conditions equals to always.
		if len(kv) != 1 {
			return false, fmt.Errorf("unexpected condition parameter")
		}

		return true, nil

	case "requestcount_lt":
		count := s.requestCount.Get(event.Context.GetID())
		if len(kv) != 2 {
			return false, fmt.Errorf("requestcount_lt condition needs a parameter")
		}
		v, err := strconv.Atoi(strings.TrimSpace(kv[1]))
		if err != nil {
			return false, fmt.Errorf("requestcount_lt condition needs an integer parameter")
		}

		if count < v {
			return true, nil
		}

	case "requestcount_gt":
		count := s.requestCount.Get(event.Context.GetID())
		if len(kv) != 2 {
			return false, fmt.Errorf("requestcount_gt condition needs a parameter")
		}
		v, err := strconv.Atoi(strings.TrimSpace(kv[1]))
		if err != nil {
			return false, fmt.Errorf("requestcount_gt condition needs an integer parameter")
		}

		if count > v {
			return true, nil
		}

	}
	return false, nil
}

func (s *StatefulHandler) executeAction(ri *ReplyInstruction, event cloudevents.Event) (*cloudevents.Event, protocol.Result) {
	kv := strings.Split(ri.Action, ":")

	switch kv[0] {
	case "delay-ack":
		if len(kv) != 2 {
			log.Print("Error evaluating action delay-ack: requires a parameter")
			return nil, protocol.ResultNACK
		}

		v, err := strconv.Atoi(strings.TrimSpace(kv[1]))
		if err != nil {
			log.Print("Error evaluating action delay-ack: requires an integer parameter")
			return nil, protocol.ResultNACK
		}

		time.Sleep(time.Duration(v) * time.Second)
		return nil, protocol.ResultACK

	case "ack":
		if len(kv) != 1 {
			log.Print("Error evaluating action ack: unexpected parameter")
			return nil, protocol.ResultNACK
		}
		return nil, protocol.ResultACK

	case "nack":
		if len(kv) != 1 {
			log.Print("Error evaluating action nack: unexpected parameter")
			return nil, protocol.ResultNACK
		}
		return nil, protocol.ResultNACK

	case "ack+payload":

	case "nack+payload":
	}

	log.Printf("Unknown action: %s", kv[0])
	return nil, protocol.ResultNACK
}
