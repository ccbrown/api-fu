package apifu

import (
	"context"
	"reflect"
)

// SubscriptionSourceStream defines the source stream for a subscription.
type SubscriptionSourceStream struct {
	// A channel of events. The channel can be of any type.
	EventChannel interface{}

	// Stop is invoked when the subscription should be stopped and the event channel should be
	// closed.
	Stop func()
}

// Run drives the stream until it's closed or until the given context is cancelled.
func (s *SubscriptionSourceStream) Run(ctx context.Context, onEvent func(interface{})) error {
	eventChannel := reflect.ValueOf(s.EventChannel)
	ctxChannel := reflect.ValueOf(ctx.Done())
	selectCases := []reflect.SelectCase{
		{
			Dir:  reflect.SelectRecv,
			Chan: ctxChannel,
		},
		{
			Dir:  reflect.SelectRecv,
			Chan: eventChannel,
		},
	}
	for {
		chosen, recv, recvOK := reflect.Select(selectCases)
		if chosen == 0 {
			// ctx.Done()
			return ctx.Err()
		}
		// s.EventChannel
		if recvOK {
			onEvent(recv.Interface())
		} else {
			return nil
		}
	}
}
