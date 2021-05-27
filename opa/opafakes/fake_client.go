// Code generated by counterfeiter. DO NOT EDIT.
package opafakes

import (
	"sync"

	"github.com/rode/rode/opa"
)

type FakeClient struct {
	EvaluatePolicyStub        func(string, []byte) (*opa.EvaluatePolicyResponse, error)
	evaluatePolicyMutex       sync.RWMutex
	evaluatePolicyArgsForCall []struct {
		arg1 string
		arg2 []byte
	}
	evaluatePolicyReturns struct {
		result1 *opa.EvaluatePolicyResponse
		result2 error
	}
	evaluatePolicyReturnsOnCall map[int]struct {
		result1 *opa.EvaluatePolicyResponse
		result2 error
	}
	InitializePolicyStub        func(string, string) opa.ClientError
	initializePolicyMutex       sync.RWMutex
	initializePolicyArgsForCall []struct {
		arg1 string
		arg2 string
	}
	initializePolicyReturns struct {
		result1 opa.ClientError
	}
	initializePolicyReturnsOnCall map[int]struct {
		result1 opa.ClientError
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeClient) EvaluatePolicy(arg1 string, arg2 []byte) (*opa.EvaluatePolicyResponse, error) {
	var arg2Copy []byte
	if arg2 != nil {
		arg2Copy = make([]byte, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.evaluatePolicyMutex.Lock()
	ret, specificReturn := fake.evaluatePolicyReturnsOnCall[len(fake.evaluatePolicyArgsForCall)]
	fake.evaluatePolicyArgsForCall = append(fake.evaluatePolicyArgsForCall, struct {
		arg1 string
		arg2 []byte
	}{arg1, arg2Copy})
	stub := fake.EvaluatePolicyStub
	fakeReturns := fake.evaluatePolicyReturns
	fake.recordInvocation("EvaluatePolicy", []interface{}{arg1, arg2Copy})
	fake.evaluatePolicyMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) EvaluatePolicyCallCount() int {
	fake.evaluatePolicyMutex.RLock()
	defer fake.evaluatePolicyMutex.RUnlock()
	return len(fake.evaluatePolicyArgsForCall)
}

func (fake *FakeClient) EvaluatePolicyCalls(stub func(string, []byte) (*opa.EvaluatePolicyResponse, error)) {
	fake.evaluatePolicyMutex.Lock()
	defer fake.evaluatePolicyMutex.Unlock()
	fake.EvaluatePolicyStub = stub
}

func (fake *FakeClient) EvaluatePolicyArgsForCall(i int) (string, []byte) {
	fake.evaluatePolicyMutex.RLock()
	defer fake.evaluatePolicyMutex.RUnlock()
	argsForCall := fake.evaluatePolicyArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeClient) EvaluatePolicyReturns(result1 *opa.EvaluatePolicyResponse, result2 error) {
	fake.evaluatePolicyMutex.Lock()
	defer fake.evaluatePolicyMutex.Unlock()
	fake.EvaluatePolicyStub = nil
	fake.evaluatePolicyReturns = struct {
		result1 *opa.EvaluatePolicyResponse
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) EvaluatePolicyReturnsOnCall(i int, result1 *opa.EvaluatePolicyResponse, result2 error) {
	fake.evaluatePolicyMutex.Lock()
	defer fake.evaluatePolicyMutex.Unlock()
	fake.EvaluatePolicyStub = nil
	if fake.evaluatePolicyReturnsOnCall == nil {
		fake.evaluatePolicyReturnsOnCall = make(map[int]struct {
			result1 *opa.EvaluatePolicyResponse
			result2 error
		})
	}
	fake.evaluatePolicyReturnsOnCall[i] = struct {
		result1 *opa.EvaluatePolicyResponse
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) InitializePolicy(arg1 string, arg2 string) opa.ClientError {
	fake.initializePolicyMutex.Lock()
	ret, specificReturn := fake.initializePolicyReturnsOnCall[len(fake.initializePolicyArgsForCall)]
	fake.initializePolicyArgsForCall = append(fake.initializePolicyArgsForCall, struct {
		arg1 string
		arg2 string
	}{arg1, arg2})
	stub := fake.InitializePolicyStub
	fakeReturns := fake.initializePolicyReturns
	fake.recordInvocation("InitializePolicy", []interface{}{arg1, arg2})
	fake.initializePolicyMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeClient) InitializePolicyCallCount() int {
	fake.initializePolicyMutex.RLock()
	defer fake.initializePolicyMutex.RUnlock()
	return len(fake.initializePolicyArgsForCall)
}

func (fake *FakeClient) InitializePolicyCalls(stub func(string, string) opa.ClientError) {
	fake.initializePolicyMutex.Lock()
	defer fake.initializePolicyMutex.Unlock()
	fake.InitializePolicyStub = stub
}

func (fake *FakeClient) InitializePolicyArgsForCall(i int) (string, string) {
	fake.initializePolicyMutex.RLock()
	defer fake.initializePolicyMutex.RUnlock()
	argsForCall := fake.initializePolicyArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeClient) InitializePolicyReturns(result1 opa.ClientError) {
	fake.initializePolicyMutex.Lock()
	defer fake.initializePolicyMutex.Unlock()
	fake.InitializePolicyStub = nil
	fake.initializePolicyReturns = struct {
		result1 opa.ClientError
	}{result1}
}

func (fake *FakeClient) InitializePolicyReturnsOnCall(i int, result1 opa.ClientError) {
	fake.initializePolicyMutex.Lock()
	defer fake.initializePolicyMutex.Unlock()
	fake.InitializePolicyStub = nil
	if fake.initializePolicyReturnsOnCall == nil {
		fake.initializePolicyReturnsOnCall = make(map[int]struct {
			result1 opa.ClientError
		})
	}
	fake.initializePolicyReturnsOnCall[i] = struct {
		result1 opa.ClientError
	}{result1}
}

func (fake *FakeClient) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.evaluatePolicyMutex.RLock()
	defer fake.evaluatePolicyMutex.RUnlock()
	fake.initializePolicyMutex.RLock()
	defer fake.initializePolicyMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeClient) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ opa.Client = new(FakeClient)