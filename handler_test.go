package main

import (
	"context"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/protocol"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

const (
	tEventID     = "test.event.id"
	tEventType   = "test.event.type"
	tEventSource = "test.event.source"
)

var (
	tReplyEventID     = tEventID + ".reply"
	tReplyEventType   = tEventType + ".reply"
	tReplyEventSource = tEventSource + ".reply"
)

type expected struct {
	event        *cloudevents.Event
	result       protocol.Result
	minimumDelay time.Duration
}

func TestHandler(t *testing.T) {
	testCases := map[string]struct {
		in *cloudevents.Event

		expected []expected

		expectEvent        *cloudevents.Event
		expectResult       protocol.Result
		expectMinimumDelay time.Duration
	}{
		"delay-ack": {
			in: newCloudEvent(withPayload(`[{"action":"delay-ack: 2"}]`)),

			expected: []expected{{
				event:        nil,
				result:       protocol.ResultACK,
				minimumDelay: 2 * time.Second,
			}},
		},

		"ack": {
			in: newCloudEvent(withPayload(`[{"action":"ack"}]`)),

			expected: []expected{{
				event:  nil,
				result: protocol.ResultACK,
			}},
		},

		"nack": {
			in: newCloudEvent(withPayload(`[{"action":"nack"}]`)),

			expected: []expected{{
				event:  nil,
				result: protocol.ResultNACK,
			}},
		},

		"ack+event": {
			in: newCloudEvent(withPayload(`[{"action":"ack+event","reply":[{"action":"ack"}]}]`)),

			expected: []expected{{
				event: newCloudEvent(
					withPayload(`[{"action":"ack"}]`),
					withEventType(tReplyEventType),
					withEventSource(tReplyEventSource),
					withID(tReplyEventID)),
				result: protocol.ResultACK,
			}},
		},

		"nack+event": {
			in: newCloudEvent(withPayload(`[{"action":"nack+event","reply":[{"discarded":"response"}]}]`)),

			expected: []expected{{
				event: newCloudEvent(
					withPayload(`[{"discarded":"response"}]`),
					withEventType(tReplyEventType),
					withEventSource(tReplyEventSource),
					withID(tReplyEventID)),
				result: protocol.ResultNACK,
			}},
		},

		"retrycount_lt 2 nack, then fallback to ack": {
			in: newCloudEvent(withPayload(`[
				{
					"condition":"retrycount_lt: 2",
					"action":"nack"
				}]`)),

			expected: []expected{
				{
					event:  nil,
					result: protocol.ResultNACK,
				},
				{
					event:  nil,
					result: protocol.ResultNACK,
				},
				{
					event:  nil,
					result: protocol.ResultACK,
				},
			},
		},

		"retrycount_gt 2 nack, else fallback to ack": {
			in: newCloudEvent(withPayload(`[
				{
					"condition":"retrycount_gt: 2",
					"action":"nack"
				}]`)),

			expected: []expected{
				{
					event:  nil,
					result: protocol.ResultACK,
				},
				{
					event:  nil,
					result: protocol.ResultACK,
				},
				{
					event:  nil,
					result: protocol.ResultACK,
				},
				{
					event:  nil,
					result: protocol.ResultNACK,
				},
			},
		},

		"unknown condition": {
			in: newCloudEvent(withPayload(`[
				{
					"condition":"unknown",
					"action":"ack"
				}]`)),

			expected: []expected{{
				event:  nil,
				result: protocol.ResultNACK,
			}},
		},

		"unknown action": {
			in: newCloudEvent(withPayload(`[
				{
					"condition":"always",
					"action":"unknown"
				}]`)),

			expected: []expected{{
				event:  nil,
				result: protocol.ResultNACK,
			}},
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			handler := RequestHandler{
				requestCount: NewRequestCountPerID(ctx, 10*time.Second, 30*time.Second),
				logger:       zaptest.NewLogger(t),
			}

			for i, e := range tc.expected {
				start := time.Now()
				out, res := handler.Handle(ctx, *tc.in)
				delay := time.Since(start)

				assert.Equal(t, e.event, out, "Iteration %d: unexpected response event", i)
				assert.Equal(t, e.result, res, "Iteration %d: non matching result", i)
				if e.minimumDelay != 0 {
					assert.GreaterOrEqual(t, delay, e.minimumDelay, "Iteration %d: delay was lower than expected", i)
				}
			}
		})
	}
}

type cloudEventOption func(*cloudevents.Event)

func newCloudEvent(opts ...cloudEventOption) *cloudevents.Event {
	out := cloudevents.NewEvent()

	out.SetID(tEventID)
	out.SetType(tEventType)
	out.SetSource(tEventSource)

	for _, f := range opts {
		f(&out)
	}

	return &out
}

func withPayload(payload string) cloudEventOption {
	return func(ce *cloudevents.Event) {
		ce.SetData(cloudevents.ApplicationJSON, []byte(payload))
	}
}

func withID(id string) cloudEventOption {
	return func(ce *cloudevents.Event) {
		ce.SetID(id)
	}
}

func withEventType(t string) cloudEventOption {
	return func(ce *cloudevents.Event) {
		ce.SetType(t)
	}
}

func withEventSource(s string) cloudEventOption {
	return func(ce *cloudevents.Event) {
		ce.SetSource(s)
	}
}
