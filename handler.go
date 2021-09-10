package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/protocol"
)

// RequestHandler dispatches requests.
type RequestHandler struct {
	// requestCount aggregates the number of calls per request ID. This enables
	// us to simulate cases where retries for the same event are needed.
	requestCount *RequestCountPerID
	logger       *zap.Logger
}

func NewRequestHandler(ctx context.Context) *RequestHandler {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339) // or time.RubyDate or "2006-01-02 15:04:05" or even freaking time.Kitchen

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	return &RequestHandler{
		requestCount: NewRequestCountPerID(ctx, 30*time.Second, 30*time.Second),
		logger:       logger,
	}
}

func (s *RequestHandler) Handle(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, protocol.Result) {
	ris := &ReplyInstructions{}
	if err := event.DataAs(ris); err != nil {
		s.logger.Info("error parsing reply instructions", zap.String("event-id", event.Context.GetID()),
			zap.String("result", "nack"), zap.Error(err))
		return nil, protocol.ResultNACK
	}

	// increase the count for the event in the request.
	s.requestCount.Increase(event.Context.GetID())

	for _, ri := range *ris {

		ok, err := s.evalCondition(&ri, event)
		if err != nil {
			s.logger.Info("error evaluating condition", zap.String("event-id", event.Context.GetID()),
				zap.String("result", "nack"), zap.Error(err))
			return nil, protocol.ResultNACK
		}

		if !ok {
			// if the condition is not true do not execute action and
			// continue processing the next instruction.
			continue
		}

		return s.executeAction(&ri, event)
	}

	s.logger.Info("fallback", zap.String("event-id", event.Context.GetID()), zap.String("result", "nack"))
	return nil, protocol.ResultACK
}

func (s *RequestHandler) evalCondition(ri *ReplyInstruction, event cloudevents.Event) (bool, error) {
	kv := strings.Split(ri.Condition, ":")

	switch kv[0] {
	case "always", "":
		// Absence of conditions equals to always.
		if len(kv) != 1 {
			return false, fmt.Errorf("unexpected condition parameter")
		}

		return true, nil

	case "retrycount_lt":
		count := s.requestCount.Get(event.Context.GetID())
		if len(kv) != 2 {
			return false, fmt.Errorf("retrycount_lt condition needs a parameter")
		}
		v, err := strconv.Atoi(strings.TrimSpace(kv[1]))
		if err != nil {
			return false, fmt.Errorf("retrycount_lt condition needs an integer parameter")
		}

		if count < v {
			return true, nil
		}

	case "retrycount_gt":
		count := s.requestCount.Get(event.Context.GetID())
		if len(kv) != 2 {
			return false, fmt.Errorf("retrycount_gt condition needs a parameter")
		}
		v, err := strconv.Atoi(strings.TrimSpace(kv[1]))
		if err != nil {
			return false, fmt.Errorf("retrycount_gt condition needs an integer parameter")
		}

		if count > v {
			return true, nil
		}

	}
	return false, nil
}

func (s *RequestHandler) executeAction(ri *ReplyInstruction, event cloudevents.Event) (*cloudevents.Event, protocol.Result) {
	kv := strings.Split(ri.Action, ":")

	switch kv[0] {
	case "delay-ack":
		if len(kv) != 2 {
			s.logger.Info("delay-ack", zap.String("event-id", event.Context.GetID()), zap.String("result", "nack"),
				zap.Error(errors.New("error evaluating action delay-ack: requires a parameter")))
			return nil, protocol.ResultNACK
		}

		v, err := strconv.Atoi(strings.TrimSpace(kv[1]))
		if err != nil {
			s.logger.Info("delay-ack", zap.String("event-id", event.Context.GetID()), zap.String("result", "nack"),
				zap.Error(errors.New("error evaluating action delay-ack: requires an integer parameter")))
			return nil, protocol.ResultNACK
		}

		time.Sleep(time.Duration(v) * time.Second)
		s.logger.Info("delay-ack", zap.String("event-id", event.Context.GetID()), zap.String("result", "ack"))
		return nil, protocol.ResultACK

	case "ack":
		if len(kv) != 1 {
			s.logger.Info("ack", zap.String("event-id", event.Context.GetID()), zap.String("result", "nack"),
				zap.Error(errors.New("error evaluating action ack: unexpected parameter")))
			return nil, protocol.ResultNACK
		}

		s.logger.Info("ack", zap.String("event-id", event.Context.GetID()), zap.String("result", "ack"))
		return nil, protocol.ResultACK

	case "nack":
		if len(kv) != 1 {
			s.logger.Info("nack", zap.String("event-id", event.Context.GetID()), zap.String("result", "nack"),
				zap.Error(errors.New("error evaluating action nack: unexpected parameter")))
			return nil, protocol.ResultNACK
		}

		s.logger.Info("nack", zap.String("event-id", event.Context.GetID()), zap.String("result", "nack"))
		return nil, protocol.ResultNACK

	case "ack+payload":
		s.logger.Info("ack+payload", zap.String("event-id", event.Context.GetID()), zap.String("result", "nack"),
			zap.Error(errors.New("not implemented")))
		return nil, protocol.ResultNACK

	case "nack+payload":
		s.logger.Info("nack+payload", zap.String("event-id", event.Context.GetID()), zap.String("result", "nack"),
			zap.Error(errors.New("not implemented")))
		return nil, protocol.ResultNACK

	case "reset":
		// Reset will delete all request counts at the memory storage.
		if len(kv) != 1 {
			s.logger.Info("reset", zap.String("event-id", event.Context.GetID()), zap.String("result", "nack"),
				zap.Error(errors.New("error evaluating action reset: unexpected parameter")))
			return nil, protocol.ResultNACK
		}

		s.logger.Debug("reset", zap.String("event-id", event.Context.GetID()), zap.String("result", "ack"))
		s.requestCount.Reset()
		return nil, protocol.ResultACK
	}

	s.logger.Info("unknown", zap.String("event-id", event.Context.GetID()), zap.String("result", "nack"),
		zap.Error(fmt.Errorf("error evaluating action: unknown action: %s", kv[0])))
	return nil, protocol.ResultNACK
}
