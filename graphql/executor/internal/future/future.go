package future

import (
	"reflect"
)

// Result holds either a value or an error.
type Result struct {
	Value interface{}
	Error error
}

// IsOk returns true if the result is not an error.
func (r Result) IsOk() bool {
	return r.Error == nil || reflect.ValueOf(r.Error).IsNil()
}

// IsErr returns true if the result is an error.
func (r Result) IsErr() bool {
	return !r.IsOk()
}

// Future represents a result that will be available at some point in the future. It is very similar
// to Rust's Future trait.
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

// MapOk converts a future's value to a different type using a conversion function.
func (f Future) MapOk(fn func(interface{}) interface{}) Future {
	if f.IsReady() {
		if f.result.IsOk() {
			f.result.Value = fn(f.result.Value)
		}
	} else {
		fpoll := f.poll
		f.poll = func() (Result, bool) {
			r, ok := fpoll()
			if ok && r.IsOk() {
				r.Value = fn(r.Value)
			}
			return r, ok
		}
	}
	return f
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

// Ok returns a new future that is immediately ready with the given value.
func Ok(v interface{}) Future {
	return Future{
		result: Result{
			Value: v,
		},
	}
}

// Err returns a new future that is immediately ready with the given error.
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

	ok := true

	for i, f := range fs {
		if f.IsReady() {
			if !f.Result().IsOk() {
				return Err(f.Result().Error)
			} else {
				results[i] = f.Result().Value
			}
		} else {
			ok = false
		}
	}

	if ok {
		return Ok(results)
	}

	return New(func() (Result, bool) {
		ok := true

		for i, f := range fs {
			f.Poll()
			if f.IsReady() {
				if !f.Result().IsOk() {
					return Result{
						Error: f.Result().Error,
					}, true
				} else {
					results[i] = f.Result().Value
				}
			} else {
				ok = false
			}
		}

		if ok {
			return Result{
				Value: results,
			}, true
		}

		return Result{}, false
	})
}

// After returns a single future that resolves after all of the given futures. If any future errors,
// the returned future immediately resolves to an error. This is very similar to Join except that
// the resolved value will be nil (making it more efficient if you don't need the values from the
// joined futures).
func After(fs ...Future) Future {
	ok := true

	for _, f := range fs {
		if f.IsReady() {
			if !f.Result().IsOk() {
				return Err(f.Result().Error)
			}
		} else {
			ok = false
		}
	}

	if ok {
		return Ok(nil)
	}

	return New(func() (Result, bool) {
		ok := true

		for _, f := range fs {
			f.Poll()
			if f.IsReady() {
				if !f.Result().IsOk() {
					return Result{
						Error: f.Result().Error,
					}, true
				}
			} else {
				ok = false
			}
		}

		if ok {
			return Result{}, true
		}

		return Result{}, false
	})
}
