package promise

import "reflect"

type Promise struct {
	isResolved bool
	value      interface{}

	isRejected bool
	err        error

	next         *Promise
	dependencies []*Promise

	source func(resolve func(interface{}), reject func(error))

	parent     *Promise
	onResolved func(value interface{}) interface{}
	onRejected func(value error) interface{}
}

// New returns a new Promise, which is very much like the JavaScript equivalent, but with one
// exception: the function given to New should not actually perform any work asynchronously (or if
// it does, it should be done transparently). The function will be invoked when the promise is
// scheduled. If the promise cannot be fulfilled yet, simply don't invoke resolve until the next
// time the function is called.
func New(f func(resolve func(interface{}), reject func(error))) *Promise {
	return &Promise{
		source: f,
	}
}

// Then appends a handler to be promise, invoking it when the promise is fulfilled. If the handler
// returns a value, it'll be passed as input to the next handler in the chain. If the handler
// returns another promise, the next handler in the chain will receive that promise's value when it
// is fulfilled.
//
// Invoking this function consumes the receiver. In fact, in many cases, the receiver is simply
// modified and returned.
func (p *Promise) Then(onResolved func(value interface{}) interface{}) *Promise {
	if p.isResolved {
		next := onResolved(p.value)
		if promise, ok := next.(*Promise); ok {
			return promise
		}
		p.value = next
		return p
	} else if p.isRejected {
		return p
	}
	newPromise := &Promise{
		parent:     p,
		onResolved: onResolved,
	}
	p.next = newPromise
	return newPromise
}

// Returns a Promise and deals with rejected cases only. The Promise returned by Catch is rejected
// if onRejected returns a Promise which is itself rejected; otherwise, it is resolved.
//
// Invoking this function consumes the receiver. In fact, in many cases, the receiver is simply
// modified and returned.
func (p *Promise) Catch(onRejected func(err error) interface{}) *Promise {
	if p.isResolved {
		return p
	} else if p.isRejected {
		next := onRejected(p.err)
		if promise, ok := next.(*Promise); ok {
			return promise
		}
		p.isRejected = false
		p.isResolved = true
		p.value = next
		return p
	}
	newPromise := &Promise{
		parent:     p,
		onRejected: onRejected,
	}
	p.next = newPromise
	return newPromise
}

// Schedule invokes pending functions for unfulfilled promises and returns true if any progress was
// made.
func (p *Promise) Schedule() (didProgress bool) {
	for i := 0; ; i++ {
		didProgress := false
		for _, dependency := range p.dependencies {
			if !dependency.isResolved {
				if dependency.Schedule() {
					didProgress = true
				}
			}
		}
		if !p.isResolved && !p.isRejected {
			if p.source != nil {
				p.source(func(value interface{}) {
					p.isResolved = true
					p.value = value
					didProgress = true
				}, func(err error) {
					p.isRejected = true
					p.err = err
					didProgress = true
				})
			} else if p.parent != nil {
				if p.parent.isResolved || p.parent.isRejected {
					if p.parent.isResolved {
						p.isResolved = true
						p.value = p.parent.value
						if p.onResolved != nil {
							p.value = p.onResolved(p.value)
						}
					} else {
						if p.onRejected != nil {
							p.isResolved = true
							p.value = p.onRejected(p.parent.err)
						} else {
							p.isRejected = true
							p.err = p.parent.err
						}
					}
					didProgress = true
					if promise, ok := p.value.(*Promise); ok {
						if p.next != nil {
							p.next.parent = promise
						}
						promise.next = p.next
						p.next = promise
					}
				} else {
					didProgress = p.parent.Schedule()
				}
			}
		}
		if p.isResolved {
			if p.next != nil {
				if p.next.Schedule() {
					didProgress = true
				}
			}
		}
		if !didProgress {
			return i > 0
		}
	}
}

// Returns a Promise object that is resolved with the given value.
func Resolve(value interface{}) *Promise {
	return &Promise{
		isResolved: true,
		value:      value,
	}
}

// Returns a Promise that is rejected with the given reason.
func Reject(reason error) *Promise {
	return &Promise{
		isRejected: true,
		err:        reason,
	}
}

// All returns a single Promise that resolves when all of the promises in the argument have resolved
// or when the iterable argument contains no promises. It rejects with the reason of the first
// promise that rejects.
func All(iterable interface{}) *Promise {
	v := reflect.ValueOf(iterable)
	result := make([]interface{}, v.Len())
	var rejectReason error
	remaining := 0
	var dependencies []*Promise
	for i := 0; i < v.Len(); i++ {
		value := v.Index(i).Interface()
		promise, ok := value.(*Promise)
		if !ok {
			result[i] = value
			continue
		} else if promise == nil {
			continue
		} else if promise.isResolved {
			result[i] = promise.value
			continue
		} else if promise.isRejected {
			return promise
		}
		i := i
		remaining++
		dependencies = append(dependencies, promise.Then(func(value interface{}) interface{} {
			result[i] = value
			remaining--
			return nil
		}).Catch(func(err error) interface{} {
			rejectReason = err
			return nil
		}))
	}
	if remaining == 0 {
		return Resolve(result)
	}
	all := New(func(resolve func(interface{}), reject func(error)) {
		if rejectReason != nil {
			reject(rejectReason)
		} else if remaining == 0 {
			resolve(result)
		}
	})
	all.dependencies = dependencies
	return all
}
