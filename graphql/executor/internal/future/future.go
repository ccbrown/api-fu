package future

import (
	"reflect"
)

type Result struct {
	Value interface{}
	Error error
}

func (r Result) IsOk() bool {
	return r.Error == nil || reflect.ValueOf(r.Error).IsNil()
}

func (r Result) IsErr() bool {
	return !r.IsOk()
}

type Future struct {
	result Result
	poll   func() (Result, bool)
}

// New constructs a new future from a poll function. When the future's value is ready, poll should
// return the value and true. Otherwise, poll should return a zero value and false.
func New(poll func() (Result, bool)) Future {
	return Future{
		poll: poll,
	}
}

// IsReady returns true if the future's value is ready.
func (f Future) IsReady() bool {
	return f.poll == nil
}

// Value returns the future's value if it is ready.
func (f Future) Result() Result {
	return f.result
}

// Map converts a future's result to a different type using a conversion function.
func (f Future) Map(fn func(Result) Result) Future {
	if f.IsReady() {
		f.result = fn(f.result)
	} else {
		fpoll := f.poll
		f.poll = func() (Result, bool) {
			r, ok := fpoll()
			if ok {
				return fn(r), true
			}
			return r, false
		}
	}
	return f
}

// Map converts a future's value to a different type using a conversion function.
func (f Future) MapOk(fn func(interface{}) interface{}) Future {
	return f.Map(func(r Result) Result {
		if r.IsOk() {
			r.Value = fn(r.Value)
		}
		return r
	})
}

// Then invokes f when the future is resolved and returns a future that resolves when f's return
// value is resolved.
func (f Future) Then(fn func(Result) Future) Future {
	if f.IsReady() {
		return fn(f.result)
	} else {
		var then Future
		var hasThen bool
		fpoll := f.poll
		f.poll = func() (Result, bool) {
			if !hasThen {
				if r, ok := fpoll(); ok {
					then = fn(r)
					hasThen = true
				}
			}
			if hasThen {
				then.Poll()
				return then.result, then.IsReady()
			}
			return Result{}, false
		}
		return f
	}
}

// Poll invokes pollers for the future and its dependencies, allowing futures to transition to
// the ready state.
func (f *Future) Poll() {
	if f.poll != nil {
		var ok bool
		if f.result, ok = f.poll(); ok {
			f.poll = nil
		}
	}
}

// Ready returns a new future that is immediately ready with the given result.
func Ready(r Result) Future {
	return Future{
		result: r,
	}
}

// Ready returns a new future that is immediately ready with the given value.
func Ok(v interface{}) Future {
	return Future{
		result: Result{
			Value: v,
		},
	}
}

// Ready returns a new future that is immediately ready with the given error.
func Err(err error) Future {
	return Future{
		result: Result{
			Error: err,
		},
	}
}

// Join combines the values from multiple futures into a single future that resolves to
// []interface{}. If any future errors, the returned future immediately resolves to an error.
func Join(fs ...Future) Future {
	results := make([]interface{}, len(fs))

	poll := func() (Result, bool) {
		ok := true
		var err error

		for i, f := range fs {
			f.Poll()
			if !f.IsReady() {
				ok = false
				break
			}
			if !f.Result().IsOk() {
				err = f.Result().Error
			} else {
				results[i] = f.Result().Value
			}
		}

		if err != nil {
			return Result{
				Error: err,
			}, true
		}

		if ok {
			return Result{
				Value: results,
			}, true
		}

		return Result{}, false
	}
	if r, ok := poll(); ok {
		return Ready(r)
	}
	return New(poll)
}
