// Code generated by go-mockgen 1.3.6; DO NOT EDIT.
//
// This file was generated by running `sg generate` (or `go-mockgen`) at the root of
// this repository. To add additional mocks to this or another package, add a new entry
// to the mockgen.yaml file in the root of this repository.

package userpasswd

import (
	"context"
	"sync"
)

// MockLockoutStore is a mock implementation of the LockoutStore interface
// (from the package
// github.com/sourcegraph/sourcegraph/cmd/frontend/internal/auth/userpasswd)
// used for unit testing.
type MockLockoutStore struct {
	// GenerateUnlockAccountURLFunc is an instance of a mock function object
	// controlling the behavior of the method GenerateUnlockAccountURL.
	GenerateUnlockAccountURLFunc *LockoutStoreGenerateUnlockAccountURLFunc
	// IncreaseFailedAttemptFunc is an instance of a mock function object
	// controlling the behavior of the method IncreaseFailedAttempt.
	IncreaseFailedAttemptFunc *LockoutStoreIncreaseFailedAttemptFunc
	// IsLockedOutFunc is an instance of a mock function object controlling
	// the behavior of the method IsLockedOut.
	IsLockedOutFunc *LockoutStoreIsLockedOutFunc
	// ResetFunc is an instance of a mock function object controlling the
	// behavior of the method Reset.
	ResetFunc *LockoutStoreResetFunc
	// SendUnlockAccountEmailFunc is an instance of a mock function object
	// controlling the behavior of the method SendUnlockAccountEmail.
	SendUnlockAccountEmailFunc *LockoutStoreSendUnlockAccountEmailFunc
	// VerifyUnlockAccountTokenAndResetFunc is an instance of a mock
	// function object controlling the behavior of the method
	// VerifyUnlockAccountTokenAndReset.
	VerifyUnlockAccountTokenAndResetFunc *LockoutStoreVerifyUnlockAccountTokenAndResetFunc
}

// NewMockLockoutStore creates a new mock of the LockoutStore interface. All
// methods return zero values for all results, unless overwritten.
func NewMockLockoutStore() *MockLockoutStore {
	return &MockLockoutStore{
		GenerateUnlockAccountURLFunc: &LockoutStoreGenerateUnlockAccountURLFunc{
			defaultHook: func(int32) (r0 string, r1 string, r2 error) {
				return
			},
		},
		IncreaseFailedAttemptFunc: &LockoutStoreIncreaseFailedAttemptFunc{
			defaultHook: func(int32) {
				return
			},
		},
		IsLockedOutFunc: &LockoutStoreIsLockedOutFunc{
			defaultHook: func(int32) (r0 string, r1 bool) {
				return
			},
		},
		ResetFunc: &LockoutStoreResetFunc{
			defaultHook: func(int32) {
				return
			},
		},
		SendUnlockAccountEmailFunc: &LockoutStoreSendUnlockAccountEmailFunc{
			defaultHook: func(context.Context, int32, string) (r0 error) {
				return
			},
		},
		VerifyUnlockAccountTokenAndResetFunc: &LockoutStoreVerifyUnlockAccountTokenAndResetFunc{
			defaultHook: func(string) (r0 bool, r1 error) {
				return
			},
		},
	}
}

// NewStrictMockLockoutStore creates a new mock of the LockoutStore
// interface. All methods panic on invocation, unless overwritten.
func NewStrictMockLockoutStore() *MockLockoutStore {
	return &MockLockoutStore{
		GenerateUnlockAccountURLFunc: &LockoutStoreGenerateUnlockAccountURLFunc{
			defaultHook: func(int32) (string, string, error) {
				panic("unexpected invocation of MockLockoutStore.GenerateUnlockAccountURL")
			},
		},
		IncreaseFailedAttemptFunc: &LockoutStoreIncreaseFailedAttemptFunc{
			defaultHook: func(int32) {
				panic("unexpected invocation of MockLockoutStore.IncreaseFailedAttempt")
			},
		},
		IsLockedOutFunc: &LockoutStoreIsLockedOutFunc{
			defaultHook: func(int32) (string, bool) {
				panic("unexpected invocation of MockLockoutStore.IsLockedOut")
			},
		},
		ResetFunc: &LockoutStoreResetFunc{
			defaultHook: func(int32) {
				panic("unexpected invocation of MockLockoutStore.Reset")
			},
		},
		SendUnlockAccountEmailFunc: &LockoutStoreSendUnlockAccountEmailFunc{
			defaultHook: func(context.Context, int32, string) error {
				panic("unexpected invocation of MockLockoutStore.SendUnlockAccountEmail")
			},
		},
		VerifyUnlockAccountTokenAndResetFunc: &LockoutStoreVerifyUnlockAccountTokenAndResetFunc{
			defaultHook: func(string) (bool, error) {
				panic("unexpected invocation of MockLockoutStore.VerifyUnlockAccountTokenAndReset")
			},
		},
	}
}

// NewMockLockoutStoreFrom creates a new mock of the MockLockoutStore
// interface. All methods delegate to the given implementation, unless
// overwritten.
func NewMockLockoutStoreFrom(i LockoutStore) *MockLockoutStore {
	return &MockLockoutStore{
		GenerateUnlockAccountURLFunc: &LockoutStoreGenerateUnlockAccountURLFunc{
			defaultHook: i.GenerateUnlockAccountURL,
		},
		IncreaseFailedAttemptFunc: &LockoutStoreIncreaseFailedAttemptFunc{
			defaultHook: i.IncreaseFailedAttempt,
		},
		IsLockedOutFunc: &LockoutStoreIsLockedOutFunc{
			defaultHook: i.IsLockedOut,
		},
		ResetFunc: &LockoutStoreResetFunc{
			defaultHook: i.Reset,
		},
		SendUnlockAccountEmailFunc: &LockoutStoreSendUnlockAccountEmailFunc{
			defaultHook: i.SendUnlockAccountEmail,
		},
		VerifyUnlockAccountTokenAndResetFunc: &LockoutStoreVerifyUnlockAccountTokenAndResetFunc{
			defaultHook: i.VerifyUnlockAccountTokenAndReset,
		},
	}
}

// LockoutStoreGenerateUnlockAccountURLFunc describes the behavior when the
// GenerateUnlockAccountURL method of the parent MockLockoutStore instance
// is invoked.
type LockoutStoreGenerateUnlockAccountURLFunc struct {
	defaultHook func(int32) (string, string, error)
	hooks       []func(int32) (string, string, error)
	history     []LockoutStoreGenerateUnlockAccountURLFuncCall
	mutex       sync.Mutex
}

// GenerateUnlockAccountURL delegates to the next hook function in the queue
// and stores the parameter and result values of this invocation.
func (m *MockLockoutStore) GenerateUnlockAccountURL(v0 int32) (string, string, error) {
	r0, r1, r2 := m.GenerateUnlockAccountURLFunc.nextHook()(v0)
	m.GenerateUnlockAccountURLFunc.appendCall(LockoutStoreGenerateUnlockAccountURLFuncCall{v0, r0, r1, r2})
	return r0, r1, r2
}

// SetDefaultHook sets function that is called when the
// GenerateUnlockAccountURL method of the parent MockLockoutStore instance
// is invoked and the hook queue is empty.
func (f *LockoutStoreGenerateUnlockAccountURLFunc) SetDefaultHook(hook func(int32) (string, string, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// GenerateUnlockAccountURL method of the parent MockLockoutStore instance
// invokes the hook at the front of the queue and discards it. After the
// queue is empty, the default hook function is invoked for any future
// action.
func (f *LockoutStoreGenerateUnlockAccountURLFunc) PushHook(hook func(int32) (string, string, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultHook with a function that returns the
// given values.
func (f *LockoutStoreGenerateUnlockAccountURLFunc) SetDefaultReturn(r0 string, r1 string, r2 error) {
	f.SetDefaultHook(func(int32) (string, string, error) {
		return r0, r1, r2
	})
}

// PushReturn calls PushHook with a function that returns the given values.
func (f *LockoutStoreGenerateUnlockAccountURLFunc) PushReturn(r0 string, r1 string, r2 error) {
	f.PushHook(func(int32) (string, string, error) {
		return r0, r1, r2
	})
}

func (f *LockoutStoreGenerateUnlockAccountURLFunc) nextHook() func(int32) (string, string, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *LockoutStoreGenerateUnlockAccountURLFunc) appendCall(r0 LockoutStoreGenerateUnlockAccountURLFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of
// LockoutStoreGenerateUnlockAccountURLFuncCall objects describing the
// invocations of this function.
func (f *LockoutStoreGenerateUnlockAccountURLFunc) History() []LockoutStoreGenerateUnlockAccountURLFuncCall {
	f.mutex.Lock()
	history := make([]LockoutStoreGenerateUnlockAccountURLFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// LockoutStoreGenerateUnlockAccountURLFuncCall is an object that describes
// an invocation of method GenerateUnlockAccountURL on an instance of
// MockLockoutStore.
type LockoutStoreGenerateUnlockAccountURLFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 int32
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 string
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 string
	// Result2 is the value of the 3rd result returned from this method
	// invocation.
	Result2 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c LockoutStoreGenerateUnlockAccountURLFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c LockoutStoreGenerateUnlockAccountURLFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1, c.Result2}
}

// LockoutStoreIncreaseFailedAttemptFunc describes the behavior when the
// IncreaseFailedAttempt method of the parent MockLockoutStore instance is
// invoked.
type LockoutStoreIncreaseFailedAttemptFunc struct {
	defaultHook func(int32)
	hooks       []func(int32)
	history     []LockoutStoreIncreaseFailedAttemptFuncCall
	mutex       sync.Mutex
}

// IncreaseFailedAttempt delegates to the next hook function in the queue
// and stores the parameter and result values of this invocation.
func (m *MockLockoutStore) IncreaseFailedAttempt(v0 int32) {
	m.IncreaseFailedAttemptFunc.nextHook()(v0)
	m.IncreaseFailedAttemptFunc.appendCall(LockoutStoreIncreaseFailedAttemptFuncCall{v0})
	return
}

// SetDefaultHook sets function that is called when the
// IncreaseFailedAttempt method of the parent MockLockoutStore instance is
// invoked and the hook queue is empty.
func (f *LockoutStoreIncreaseFailedAttemptFunc) SetDefaultHook(hook func(int32)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// IncreaseFailedAttempt method of the parent MockLockoutStore instance
// invokes the hook at the front of the queue and discards it. After the
// queue is empty, the default hook function is invoked for any future
// action.
func (f *LockoutStoreIncreaseFailedAttemptFunc) PushHook(hook func(int32)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultHook with a function that returns the
// given values.
func (f *LockoutStoreIncreaseFailedAttemptFunc) SetDefaultReturn() {
	f.SetDefaultHook(func(int32) {
		return
	})
}

// PushReturn calls PushHook with a function that returns the given values.
func (f *LockoutStoreIncreaseFailedAttemptFunc) PushReturn() {
	f.PushHook(func(int32) {
		return
	})
}

func (f *LockoutStoreIncreaseFailedAttemptFunc) nextHook() func(int32) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *LockoutStoreIncreaseFailedAttemptFunc) appendCall(r0 LockoutStoreIncreaseFailedAttemptFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of LockoutStoreIncreaseFailedAttemptFuncCall
// objects describing the invocations of this function.
func (f *LockoutStoreIncreaseFailedAttemptFunc) History() []LockoutStoreIncreaseFailedAttemptFuncCall {
	f.mutex.Lock()
	history := make([]LockoutStoreIncreaseFailedAttemptFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// LockoutStoreIncreaseFailedAttemptFuncCall is an object that describes an
// invocation of method IncreaseFailedAttempt on an instance of
// MockLockoutStore.
type LockoutStoreIncreaseFailedAttemptFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 int32
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c LockoutStoreIncreaseFailedAttemptFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c LockoutStoreIncreaseFailedAttemptFuncCall) Results() []interface{} {
	return []interface{}{}
}

// LockoutStoreIsLockedOutFunc describes the behavior when the IsLockedOut
// method of the parent MockLockoutStore instance is invoked.
type LockoutStoreIsLockedOutFunc struct {
	defaultHook func(int32) (string, bool)
	hooks       []func(int32) (string, bool)
	history     []LockoutStoreIsLockedOutFuncCall
	mutex       sync.Mutex
}

// IsLockedOut delegates to the next hook function in the queue and stores
// the parameter and result values of this invocation.
func (m *MockLockoutStore) IsLockedOut(v0 int32) (string, bool) {
	r0, r1 := m.IsLockedOutFunc.nextHook()(v0)
	m.IsLockedOutFunc.appendCall(LockoutStoreIsLockedOutFuncCall{v0, r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the IsLockedOut method
// of the parent MockLockoutStore instance is invoked and the hook queue is
// empty.
func (f *LockoutStoreIsLockedOutFunc) SetDefaultHook(hook func(int32) (string, bool)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// IsLockedOut method of the parent MockLockoutStore instance invokes the
// hook at the front of the queue and discards it. After the queue is empty,
// the default hook function is invoked for any future action.
func (f *LockoutStoreIsLockedOutFunc) PushHook(hook func(int32) (string, bool)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultHook with a function that returns the
// given values.
func (f *LockoutStoreIsLockedOutFunc) SetDefaultReturn(r0 string, r1 bool) {
	f.SetDefaultHook(func(int32) (string, bool) {
		return r0, r1
	})
}

// PushReturn calls PushHook with a function that returns the given values.
func (f *LockoutStoreIsLockedOutFunc) PushReturn(r0 string, r1 bool) {
	f.PushHook(func(int32) (string, bool) {
		return r0, r1
	})
}

func (f *LockoutStoreIsLockedOutFunc) nextHook() func(int32) (string, bool) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *LockoutStoreIsLockedOutFunc) appendCall(r0 LockoutStoreIsLockedOutFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of LockoutStoreIsLockedOutFuncCall objects
// describing the invocations of this function.
func (f *LockoutStoreIsLockedOutFunc) History() []LockoutStoreIsLockedOutFuncCall {
	f.mutex.Lock()
	history := make([]LockoutStoreIsLockedOutFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// LockoutStoreIsLockedOutFuncCall is an object that describes an invocation
// of method IsLockedOut on an instance of MockLockoutStore.
type LockoutStoreIsLockedOutFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 int32
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 string
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 bool
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c LockoutStoreIsLockedOutFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c LockoutStoreIsLockedOutFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}

// LockoutStoreResetFunc describes the behavior when the Reset method of the
// parent MockLockoutStore instance is invoked.
type LockoutStoreResetFunc struct {
	defaultHook func(int32)
	hooks       []func(int32)
	history     []LockoutStoreResetFuncCall
	mutex       sync.Mutex
}

// Reset delegates to the next hook function in the queue and stores the
// parameter and result values of this invocation.
func (m *MockLockoutStore) Reset(v0 int32) {
	m.ResetFunc.nextHook()(v0)
	m.ResetFunc.appendCall(LockoutStoreResetFuncCall{v0})
	return
}

// SetDefaultHook sets function that is called when the Reset method of the
// parent MockLockoutStore instance is invoked and the hook queue is empty.
func (f *LockoutStoreResetFunc) SetDefaultHook(hook func(int32)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// Reset method of the parent MockLockoutStore instance invokes the hook at
// the front of the queue and discards it. After the queue is empty, the
// default hook function is invoked for any future action.
func (f *LockoutStoreResetFunc) PushHook(hook func(int32)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultHook with a function that returns the
// given values.
func (f *LockoutStoreResetFunc) SetDefaultReturn() {
	f.SetDefaultHook(func(int32) {
		return
	})
}

// PushReturn calls PushHook with a function that returns the given values.
func (f *LockoutStoreResetFunc) PushReturn() {
	f.PushHook(func(int32) {
		return
	})
}

func (f *LockoutStoreResetFunc) nextHook() func(int32) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *LockoutStoreResetFunc) appendCall(r0 LockoutStoreResetFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of LockoutStoreResetFuncCall objects
// describing the invocations of this function.
func (f *LockoutStoreResetFunc) History() []LockoutStoreResetFuncCall {
	f.mutex.Lock()
	history := make([]LockoutStoreResetFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// LockoutStoreResetFuncCall is an object that describes an invocation of
// method Reset on an instance of MockLockoutStore.
type LockoutStoreResetFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 int32
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c LockoutStoreResetFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c LockoutStoreResetFuncCall) Results() []interface{} {
	return []interface{}{}
}

// LockoutStoreSendUnlockAccountEmailFunc describes the behavior when the
// SendUnlockAccountEmail method of the parent MockLockoutStore instance is
// invoked.
type LockoutStoreSendUnlockAccountEmailFunc struct {
	defaultHook func(context.Context, int32, string) error
	hooks       []func(context.Context, int32, string) error
	history     []LockoutStoreSendUnlockAccountEmailFuncCall
	mutex       sync.Mutex
}

// SendUnlockAccountEmail delegates to the next hook function in the queue
// and stores the parameter and result values of this invocation.
func (m *MockLockoutStore) SendUnlockAccountEmail(v0 context.Context, v1 int32, v2 string) error {
	r0 := m.SendUnlockAccountEmailFunc.nextHook()(v0, v1, v2)
	m.SendUnlockAccountEmailFunc.appendCall(LockoutStoreSendUnlockAccountEmailFuncCall{v0, v1, v2, r0})
	return r0
}

// SetDefaultHook sets function that is called when the
// SendUnlockAccountEmail method of the parent MockLockoutStore instance is
// invoked and the hook queue is empty.
func (f *LockoutStoreSendUnlockAccountEmailFunc) SetDefaultHook(hook func(context.Context, int32, string) error) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// SendUnlockAccountEmail method of the parent MockLockoutStore instance
// invokes the hook at the front of the queue and discards it. After the
// queue is empty, the default hook function is invoked for any future
// action.
func (f *LockoutStoreSendUnlockAccountEmailFunc) PushHook(hook func(context.Context, int32, string) error) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultHook with a function that returns the
// given values.
func (f *LockoutStoreSendUnlockAccountEmailFunc) SetDefaultReturn(r0 error) {
	f.SetDefaultHook(func(context.Context, int32, string) error {
		return r0
	})
}

// PushReturn calls PushHook with a function that returns the given values.
func (f *LockoutStoreSendUnlockAccountEmailFunc) PushReturn(r0 error) {
	f.PushHook(func(context.Context, int32, string) error {
		return r0
	})
}

func (f *LockoutStoreSendUnlockAccountEmailFunc) nextHook() func(context.Context, int32, string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *LockoutStoreSendUnlockAccountEmailFunc) appendCall(r0 LockoutStoreSendUnlockAccountEmailFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of LockoutStoreSendUnlockAccountEmailFuncCall
// objects describing the invocations of this function.
func (f *LockoutStoreSendUnlockAccountEmailFunc) History() []LockoutStoreSendUnlockAccountEmailFuncCall {
	f.mutex.Lock()
	history := make([]LockoutStoreSendUnlockAccountEmailFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// LockoutStoreSendUnlockAccountEmailFuncCall is an object that describes an
// invocation of method SendUnlockAccountEmail on an instance of
// MockLockoutStore.
type LockoutStoreSendUnlockAccountEmailFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 int32
	// Arg2 is the value of the 3rd argument passed to this method
	// invocation.
	Arg2 string
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c LockoutStoreSendUnlockAccountEmailFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1, c.Arg2}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c LockoutStoreSendUnlockAccountEmailFuncCall) Results() []interface{} {
	return []interface{}{c.Result0}
}

// LockoutStoreVerifyUnlockAccountTokenAndResetFunc describes the behavior
// when the VerifyUnlockAccountTokenAndReset method of the parent
// MockLockoutStore instance is invoked.
type LockoutStoreVerifyUnlockAccountTokenAndResetFunc struct {
	defaultHook func(string) (bool, error)
	hooks       []func(string) (bool, error)
	history     []LockoutStoreVerifyUnlockAccountTokenAndResetFuncCall
	mutex       sync.Mutex
}

// VerifyUnlockAccountTokenAndReset delegates to the next hook function in
// the queue and stores the parameter and result values of this invocation.
func (m *MockLockoutStore) VerifyUnlockAccountTokenAndReset(v0 string) (bool, error) {
	r0, r1 := m.VerifyUnlockAccountTokenAndResetFunc.nextHook()(v0)
	m.VerifyUnlockAccountTokenAndResetFunc.appendCall(LockoutStoreVerifyUnlockAccountTokenAndResetFuncCall{v0, r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the
// VerifyUnlockAccountTokenAndReset method of the parent MockLockoutStore
// instance is invoked and the hook queue is empty.
func (f *LockoutStoreVerifyUnlockAccountTokenAndResetFunc) SetDefaultHook(hook func(string) (bool, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// VerifyUnlockAccountTokenAndReset method of the parent MockLockoutStore
// instance invokes the hook at the front of the queue and discards it.
// After the queue is empty, the default hook function is invoked for any
// future action.
func (f *LockoutStoreVerifyUnlockAccountTokenAndResetFunc) PushHook(hook func(string) (bool, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultHook with a function that returns the
// given values.
func (f *LockoutStoreVerifyUnlockAccountTokenAndResetFunc) SetDefaultReturn(r0 bool, r1 error) {
	f.SetDefaultHook(func(string) (bool, error) {
		return r0, r1
	})
}

// PushReturn calls PushHook with a function that returns the given values.
func (f *LockoutStoreVerifyUnlockAccountTokenAndResetFunc) PushReturn(r0 bool, r1 error) {
	f.PushHook(func(string) (bool, error) {
		return r0, r1
	})
}

func (f *LockoutStoreVerifyUnlockAccountTokenAndResetFunc) nextHook() func(string) (bool, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *LockoutStoreVerifyUnlockAccountTokenAndResetFunc) appendCall(r0 LockoutStoreVerifyUnlockAccountTokenAndResetFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of
// LockoutStoreVerifyUnlockAccountTokenAndResetFuncCall objects describing
// the invocations of this function.
func (f *LockoutStoreVerifyUnlockAccountTokenAndResetFunc) History() []LockoutStoreVerifyUnlockAccountTokenAndResetFuncCall {
	f.mutex.Lock()
	history := make([]LockoutStoreVerifyUnlockAccountTokenAndResetFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// LockoutStoreVerifyUnlockAccountTokenAndResetFuncCall is an object that
// describes an invocation of method VerifyUnlockAccountTokenAndReset on an
// instance of MockLockoutStore.
type LockoutStoreVerifyUnlockAccountTokenAndResetFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 string
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 bool
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c LockoutStoreVerifyUnlockAccountTokenAndResetFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c LockoutStoreVerifyUnlockAccountTokenAndResetFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}
