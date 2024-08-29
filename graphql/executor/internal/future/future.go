package future

import (
	"reflect"
)

// Result holds either a value or an error.
type Result[T any] struct {
	Value T
	Error error
}

// IsOk returns true if the result is not an error.
func (r Result[T]) IsOk() bool {
	return r.Error == nil || reflect.ValueOf(r.Error).IsNil()
}

// IsErr returns true if the result is an error.
func (r Result[T]) IsErr() bool {
	return !r.IsOk()
}

// Future represents a result that will be available at some point in the future. It is very similar
// to Rust's Future trait.
type Future[T any] struct {
	result Result[T]
	poll   func() (Result[T], bool)
}

// New constructs a new future from a poll function. When the future's value is ready, poll should
// return the value and true. Otherwise, poll should return a zero value and false.
func New[T any](poll func() (Result[T], bool)) Future[T] {
	return Future[T]{
		poll: poll,
	}
}

// IsReady returns true if the future's value is ready.
func (f Future[T]) IsReady() bool {
	return f.poll == nil
}

// Result returns the future's result if it is ready.
func (f Future[T]) Result() Result[T] {
	return f.result
}

// Map converts a future's result to a different type using a conversion function.
func Map[T any, U any](f Future[T], fn func(Result[T]) Result[U]) Future[U] {
	if f.IsReady() {
		return Future[U]{
			result: fn(f.result),
		}
	} else {
		return Future[U]{
			poll: func() (Result[U], bool) {
				r, ok := f.poll()
				if ok {
					return fn(r), true
				}
				var r2 Result[U]
				return r2, false
			},
		}
	}
}

// MapOk converts a future's value to a different type using a conversion function.
func MapOk[T any, U any](f Future[T], fn func(T) U) Future[U] {
	if f.IsReady() {
		var r Result[U]
		if f.result.IsOk() {
			r.Value = fn(f.result.Value)
		} else {
			r.Error = f.result.Error
		}
		return Future[U]{
			result: r,
		}
	} else {
		return Future[U]{
			poll: func() (Result[U], bool) {
				r, ok := f.poll()
				var r2 Result[U]
				if ok && r.IsOk() {
					r2.Value = fn(r.Value)
				}
				return r2, ok
			},
		}
	}
}

// MapOk converts a future's value to an `any` type.
func MapOkToAny[T any](f Future[T]) Future[any] {
	if f.IsReady() {
		var r Result[any]
		if f.result.IsOk() {
			r.Value = f.result.Value
		} else {
			r.Error = f.result.Error
		}
		return Future[any]{
			result: r,
		}
	} else {
		return Future[any]{
			poll: func() (Result[any], bool) {
				r, ok := f.poll()
				var r2 Result[any]
				if ok && r.IsOk() {
					r2.Value = r.Value
				}
				return r2, ok
			},
		}
	}
}

// MapOk converts a future's value to a value of a different type.
func MapOkValue[T any, U any](f Future[T], v U) Future[U] {
	if f.IsReady() {
		var r Result[U]
		if f.result.IsOk() {
			r.Value = v
		} else {
			r.Error = f.result.Error
		}
		return Future[U]{
			result: r,
		}
	} else {
		return Future[U]{
			poll: func() (Result[U], bool) {
				r, ok := f.poll()
				var r2 Result[U]
				if ok && r.IsOk() {
					r2.Value = v
				}
				return r2, ok
			},
		}
	}
}

// Then invokes f when the future is resolved and returns a future that resolves when f's return
// value is resolved.
func Then[T any, U any](f Future[T], fn func(Result[T]) Future[U]) Future[U] {
	if f.IsReady() {
		return fn(f.result)
	}
	var then Future[U]
	var hasThen bool
	fpoll := f.poll
	return Future[U]{
		poll: func() (Result[U], bool) {
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
			return Result[U]{}, false
		},
	}
}

// Poll invokes pollers for the future and its dependencies, allowing futures to transition to
// the ready state.
func (f *Future[T]) Poll() {
	if f.poll != nil {
		var ok bool
		if f.result, ok = f.poll(); ok {
			f.poll = nil
		}
	}
}

// Ok returns a new future that is immediately ready with the given value.
func Ok[T any](v T) Future[T] {
	return Future[T]{
		result: Result[T]{
			Value: v,
		},
	}
}

// Err returns a new future that is immediately ready with the given error.
func Err[T any](err error) Future[T] {
	return Future[T]{
		result: Result[T]{
			Error: err,
		},
	}
}

// Join combines the values from multiple futures into a single future that resolves to
// []T. If any future errors, the returned future immediately resolves to an error.
func Join[T any](fs ...Future[T]) Future[[]T] {
	results := make([]T, len(fs))

	ok := true

	for i, f := range fs {
		if f.IsReady() {
			if !f.Result().IsOk() {
				return Err[[]T](f.Result().Error)
			}
			results[i] = f.Result().Value
		} else {
			ok = false
		}
	}

	if ok {
		return Ok(results)
	}

	return New(func() (Result[[]T], bool) {
		ok := true

		for i := range fs {
			f := &fs[i]
			f.Poll()
			if f.IsReady() {
				if !f.Result().IsOk() {
					return Result[[]T]{
						Error: f.Result().Error,
					}, true
				}
				results[i] = f.Result().Value
			} else {
				ok = false
			}
		}

		if ok {
			return Result[[]T]{
				Value: results,
			}, true
		}

		return Result[[]T]{}, false
	})
}

// After returns a single future that resolves after all of the given futures. If any future errors,
// the returned future immediately resolves to an error. This is very similar to Join except that
// the resolved value will be empty (making it more efficient if you don't need the values from the
// joined futures).
func After[T any](fs ...Future[T]) Future[struct{}] {
	ok := true

	for _, f := range fs {
		if f.IsReady() {
			if !f.Result().IsOk() {
				return Err[struct{}](f.Result().Error)
			}
		} else {
			ok = false
		}
	}

	if ok {
		return Ok(struct{}{})
	}

	return New(func() (Result[struct{}], bool) {
		ok := true

		for _, f := range fs {
			f.Poll()
			if f.IsReady() {
				if !f.Result().IsOk() {
					return Result[struct{}]{
						Error: f.Result().Error,
					}, true
				}
			} else {
				ok = false
			}
		}

		if ok {
			return Result[struct{}]{}, true
		}

		return Result[struct{}]{}, false
	})
}
