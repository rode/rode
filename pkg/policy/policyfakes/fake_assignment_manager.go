// Code generated by counterfeiter. DO NOT EDIT.
package policyfakes

import (
	"context"
	"sync"

	"github.com/rode/rode/pkg/policy"
	"github.com/rode/rode/proto/v1alpha1"
	"google.golang.org/protobuf/types/known/emptypb"
)

type FakeAssignmentManager struct {
	CreatePolicyAssignmentStub        func(context.Context, *v1alpha1.PolicyAssignment) (*v1alpha1.PolicyAssignment, error)
	createPolicyAssignmentMutex       sync.RWMutex
	createPolicyAssignmentArgsForCall []struct {
		arg1 context.Context
		arg2 *v1alpha1.PolicyAssignment
	}
	createPolicyAssignmentReturns struct {
		result1 *v1alpha1.PolicyAssignment
		result2 error
	}
	createPolicyAssignmentReturnsOnCall map[int]struct {
		result1 *v1alpha1.PolicyAssignment
		result2 error
	}
	DeletePolicyAssignmentStub        func(context.Context, *v1alpha1.DeletePolicyAssignmentRequest) (*emptypb.Empty, error)
	deletePolicyAssignmentMutex       sync.RWMutex
	deletePolicyAssignmentArgsForCall []struct {
		arg1 context.Context
		arg2 *v1alpha1.DeletePolicyAssignmentRequest
	}
	deletePolicyAssignmentReturns struct {
		result1 *emptypb.Empty
		result2 error
	}
	deletePolicyAssignmentReturnsOnCall map[int]struct {
		result1 *emptypb.Empty
		result2 error
	}
	GetPolicyAssignmentStub        func(context.Context, *v1alpha1.GetPolicyAssignmentRequest) (*v1alpha1.PolicyAssignment, error)
	getPolicyAssignmentMutex       sync.RWMutex
	getPolicyAssignmentArgsForCall []struct {
		arg1 context.Context
		arg2 *v1alpha1.GetPolicyAssignmentRequest
	}
	getPolicyAssignmentReturns struct {
		result1 *v1alpha1.PolicyAssignment
		result2 error
	}
	getPolicyAssignmentReturnsOnCall map[int]struct {
		result1 *v1alpha1.PolicyAssignment
		result2 error
	}
	ListPolicyAssignmentsStub        func(context.Context, *v1alpha1.ListPolicyAssignmentsRequest) (*v1alpha1.ListPolicyAssignmentsResponse, error)
	listPolicyAssignmentsMutex       sync.RWMutex
	listPolicyAssignmentsArgsForCall []struct {
		arg1 context.Context
		arg2 *v1alpha1.ListPolicyAssignmentsRequest
	}
	listPolicyAssignmentsReturns struct {
		result1 *v1alpha1.ListPolicyAssignmentsResponse
		result2 error
	}
	listPolicyAssignmentsReturnsOnCall map[int]struct {
		result1 *v1alpha1.ListPolicyAssignmentsResponse
		result2 error
	}
	UpdatePolicyAssignmentStub        func(context.Context, *v1alpha1.PolicyAssignment) (*v1alpha1.PolicyAssignment, error)
	updatePolicyAssignmentMutex       sync.RWMutex
	updatePolicyAssignmentArgsForCall []struct {
		arg1 context.Context
		arg2 *v1alpha1.PolicyAssignment
	}
	updatePolicyAssignmentReturns struct {
		result1 *v1alpha1.PolicyAssignment
		result2 error
	}
	updatePolicyAssignmentReturnsOnCall map[int]struct {
		result1 *v1alpha1.PolicyAssignment
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeAssignmentManager) CreatePolicyAssignment(arg1 context.Context, arg2 *v1alpha1.PolicyAssignment) (*v1alpha1.PolicyAssignment, error) {
	fake.createPolicyAssignmentMutex.Lock()
	ret, specificReturn := fake.createPolicyAssignmentReturnsOnCall[len(fake.createPolicyAssignmentArgsForCall)]
	fake.createPolicyAssignmentArgsForCall = append(fake.createPolicyAssignmentArgsForCall, struct {
		arg1 context.Context
		arg2 *v1alpha1.PolicyAssignment
	}{arg1, arg2})
	stub := fake.CreatePolicyAssignmentStub
	fakeReturns := fake.createPolicyAssignmentReturns
	fake.recordInvocation("CreatePolicyAssignment", []interface{}{arg1, arg2})
	fake.createPolicyAssignmentMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeAssignmentManager) CreatePolicyAssignmentCallCount() int {
	fake.createPolicyAssignmentMutex.RLock()
	defer fake.createPolicyAssignmentMutex.RUnlock()
	return len(fake.createPolicyAssignmentArgsForCall)
}

func (fake *FakeAssignmentManager) CreatePolicyAssignmentCalls(stub func(context.Context, *v1alpha1.PolicyAssignment) (*v1alpha1.PolicyAssignment, error)) {
	fake.createPolicyAssignmentMutex.Lock()
	defer fake.createPolicyAssignmentMutex.Unlock()
	fake.CreatePolicyAssignmentStub = stub
}

func (fake *FakeAssignmentManager) CreatePolicyAssignmentArgsForCall(i int) (context.Context, *v1alpha1.PolicyAssignment) {
	fake.createPolicyAssignmentMutex.RLock()
	defer fake.createPolicyAssignmentMutex.RUnlock()
	argsForCall := fake.createPolicyAssignmentArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeAssignmentManager) CreatePolicyAssignmentReturns(result1 *v1alpha1.PolicyAssignment, result2 error) {
	fake.createPolicyAssignmentMutex.Lock()
	defer fake.createPolicyAssignmentMutex.Unlock()
	fake.CreatePolicyAssignmentStub = nil
	fake.createPolicyAssignmentReturns = struct {
		result1 *v1alpha1.PolicyAssignment
		result2 error
	}{result1, result2}
}

func (fake *FakeAssignmentManager) CreatePolicyAssignmentReturnsOnCall(i int, result1 *v1alpha1.PolicyAssignment, result2 error) {
	fake.createPolicyAssignmentMutex.Lock()
	defer fake.createPolicyAssignmentMutex.Unlock()
	fake.CreatePolicyAssignmentStub = nil
	if fake.createPolicyAssignmentReturnsOnCall == nil {
		fake.createPolicyAssignmentReturnsOnCall = make(map[int]struct {
			result1 *v1alpha1.PolicyAssignment
			result2 error
		})
	}
	fake.createPolicyAssignmentReturnsOnCall[i] = struct {
		result1 *v1alpha1.PolicyAssignment
		result2 error
	}{result1, result2}
}

func (fake *FakeAssignmentManager) DeletePolicyAssignment(arg1 context.Context, arg2 *v1alpha1.DeletePolicyAssignmentRequest) (*emptypb.Empty, error) {
	fake.deletePolicyAssignmentMutex.Lock()
	ret, specificReturn := fake.deletePolicyAssignmentReturnsOnCall[len(fake.deletePolicyAssignmentArgsForCall)]
	fake.deletePolicyAssignmentArgsForCall = append(fake.deletePolicyAssignmentArgsForCall, struct {
		arg1 context.Context
		arg2 *v1alpha1.DeletePolicyAssignmentRequest
	}{arg1, arg2})
	stub := fake.DeletePolicyAssignmentStub
	fakeReturns := fake.deletePolicyAssignmentReturns
	fake.recordInvocation("DeletePolicyAssignment", []interface{}{arg1, arg2})
	fake.deletePolicyAssignmentMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeAssignmentManager) DeletePolicyAssignmentCallCount() int {
	fake.deletePolicyAssignmentMutex.RLock()
	defer fake.deletePolicyAssignmentMutex.RUnlock()
	return len(fake.deletePolicyAssignmentArgsForCall)
}

func (fake *FakeAssignmentManager) DeletePolicyAssignmentCalls(stub func(context.Context, *v1alpha1.DeletePolicyAssignmentRequest) (*emptypb.Empty, error)) {
	fake.deletePolicyAssignmentMutex.Lock()
	defer fake.deletePolicyAssignmentMutex.Unlock()
	fake.DeletePolicyAssignmentStub = stub
}

func (fake *FakeAssignmentManager) DeletePolicyAssignmentArgsForCall(i int) (context.Context, *v1alpha1.DeletePolicyAssignmentRequest) {
	fake.deletePolicyAssignmentMutex.RLock()
	defer fake.deletePolicyAssignmentMutex.RUnlock()
	argsForCall := fake.deletePolicyAssignmentArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeAssignmentManager) DeletePolicyAssignmentReturns(result1 *emptypb.Empty, result2 error) {
	fake.deletePolicyAssignmentMutex.Lock()
	defer fake.deletePolicyAssignmentMutex.Unlock()
	fake.DeletePolicyAssignmentStub = nil
	fake.deletePolicyAssignmentReturns = struct {
		result1 *emptypb.Empty
		result2 error
	}{result1, result2}
}

func (fake *FakeAssignmentManager) DeletePolicyAssignmentReturnsOnCall(i int, result1 *emptypb.Empty, result2 error) {
	fake.deletePolicyAssignmentMutex.Lock()
	defer fake.deletePolicyAssignmentMutex.Unlock()
	fake.DeletePolicyAssignmentStub = nil
	if fake.deletePolicyAssignmentReturnsOnCall == nil {
		fake.deletePolicyAssignmentReturnsOnCall = make(map[int]struct {
			result1 *emptypb.Empty
			result2 error
		})
	}
	fake.deletePolicyAssignmentReturnsOnCall[i] = struct {
		result1 *emptypb.Empty
		result2 error
	}{result1, result2}
}

func (fake *FakeAssignmentManager) GetPolicyAssignment(arg1 context.Context, arg2 *v1alpha1.GetPolicyAssignmentRequest) (*v1alpha1.PolicyAssignment, error) {
	fake.getPolicyAssignmentMutex.Lock()
	ret, specificReturn := fake.getPolicyAssignmentReturnsOnCall[len(fake.getPolicyAssignmentArgsForCall)]
	fake.getPolicyAssignmentArgsForCall = append(fake.getPolicyAssignmentArgsForCall, struct {
		arg1 context.Context
		arg2 *v1alpha1.GetPolicyAssignmentRequest
	}{arg1, arg2})
	stub := fake.GetPolicyAssignmentStub
	fakeReturns := fake.getPolicyAssignmentReturns
	fake.recordInvocation("GetPolicyAssignment", []interface{}{arg1, arg2})
	fake.getPolicyAssignmentMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeAssignmentManager) GetPolicyAssignmentCallCount() int {
	fake.getPolicyAssignmentMutex.RLock()
	defer fake.getPolicyAssignmentMutex.RUnlock()
	return len(fake.getPolicyAssignmentArgsForCall)
}

func (fake *FakeAssignmentManager) GetPolicyAssignmentCalls(stub func(context.Context, *v1alpha1.GetPolicyAssignmentRequest) (*v1alpha1.PolicyAssignment, error)) {
	fake.getPolicyAssignmentMutex.Lock()
	defer fake.getPolicyAssignmentMutex.Unlock()
	fake.GetPolicyAssignmentStub = stub
}

func (fake *FakeAssignmentManager) GetPolicyAssignmentArgsForCall(i int) (context.Context, *v1alpha1.GetPolicyAssignmentRequest) {
	fake.getPolicyAssignmentMutex.RLock()
	defer fake.getPolicyAssignmentMutex.RUnlock()
	argsForCall := fake.getPolicyAssignmentArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeAssignmentManager) GetPolicyAssignmentReturns(result1 *v1alpha1.PolicyAssignment, result2 error) {
	fake.getPolicyAssignmentMutex.Lock()
	defer fake.getPolicyAssignmentMutex.Unlock()
	fake.GetPolicyAssignmentStub = nil
	fake.getPolicyAssignmentReturns = struct {
		result1 *v1alpha1.PolicyAssignment
		result2 error
	}{result1, result2}
}

func (fake *FakeAssignmentManager) GetPolicyAssignmentReturnsOnCall(i int, result1 *v1alpha1.PolicyAssignment, result2 error) {
	fake.getPolicyAssignmentMutex.Lock()
	defer fake.getPolicyAssignmentMutex.Unlock()
	fake.GetPolicyAssignmentStub = nil
	if fake.getPolicyAssignmentReturnsOnCall == nil {
		fake.getPolicyAssignmentReturnsOnCall = make(map[int]struct {
			result1 *v1alpha1.PolicyAssignment
			result2 error
		})
	}
	fake.getPolicyAssignmentReturnsOnCall[i] = struct {
		result1 *v1alpha1.PolicyAssignment
		result2 error
	}{result1, result2}
}

func (fake *FakeAssignmentManager) ListPolicyAssignments(arg1 context.Context, arg2 *v1alpha1.ListPolicyAssignmentsRequest) (*v1alpha1.ListPolicyAssignmentsResponse, error) {
	fake.listPolicyAssignmentsMutex.Lock()
	ret, specificReturn := fake.listPolicyAssignmentsReturnsOnCall[len(fake.listPolicyAssignmentsArgsForCall)]
	fake.listPolicyAssignmentsArgsForCall = append(fake.listPolicyAssignmentsArgsForCall, struct {
		arg1 context.Context
		arg2 *v1alpha1.ListPolicyAssignmentsRequest
	}{arg1, arg2})
	stub := fake.ListPolicyAssignmentsStub
	fakeReturns := fake.listPolicyAssignmentsReturns
	fake.recordInvocation("ListPolicyAssignments", []interface{}{arg1, arg2})
	fake.listPolicyAssignmentsMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeAssignmentManager) ListPolicyAssignmentsCallCount() int {
	fake.listPolicyAssignmentsMutex.RLock()
	defer fake.listPolicyAssignmentsMutex.RUnlock()
	return len(fake.listPolicyAssignmentsArgsForCall)
}

func (fake *FakeAssignmentManager) ListPolicyAssignmentsCalls(stub func(context.Context, *v1alpha1.ListPolicyAssignmentsRequest) (*v1alpha1.ListPolicyAssignmentsResponse, error)) {
	fake.listPolicyAssignmentsMutex.Lock()
	defer fake.listPolicyAssignmentsMutex.Unlock()
	fake.ListPolicyAssignmentsStub = stub
}

func (fake *FakeAssignmentManager) ListPolicyAssignmentsArgsForCall(i int) (context.Context, *v1alpha1.ListPolicyAssignmentsRequest) {
	fake.listPolicyAssignmentsMutex.RLock()
	defer fake.listPolicyAssignmentsMutex.RUnlock()
	argsForCall := fake.listPolicyAssignmentsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeAssignmentManager) ListPolicyAssignmentsReturns(result1 *v1alpha1.ListPolicyAssignmentsResponse, result2 error) {
	fake.listPolicyAssignmentsMutex.Lock()
	defer fake.listPolicyAssignmentsMutex.Unlock()
	fake.ListPolicyAssignmentsStub = nil
	fake.listPolicyAssignmentsReturns = struct {
		result1 *v1alpha1.ListPolicyAssignmentsResponse
		result2 error
	}{result1, result2}
}

func (fake *FakeAssignmentManager) ListPolicyAssignmentsReturnsOnCall(i int, result1 *v1alpha1.ListPolicyAssignmentsResponse, result2 error) {
	fake.listPolicyAssignmentsMutex.Lock()
	defer fake.listPolicyAssignmentsMutex.Unlock()
	fake.ListPolicyAssignmentsStub = nil
	if fake.listPolicyAssignmentsReturnsOnCall == nil {
		fake.listPolicyAssignmentsReturnsOnCall = make(map[int]struct {
			result1 *v1alpha1.ListPolicyAssignmentsResponse
			result2 error
		})
	}
	fake.listPolicyAssignmentsReturnsOnCall[i] = struct {
		result1 *v1alpha1.ListPolicyAssignmentsResponse
		result2 error
	}{result1, result2}
}

func (fake *FakeAssignmentManager) UpdatePolicyAssignment(arg1 context.Context, arg2 *v1alpha1.PolicyAssignment) (*v1alpha1.PolicyAssignment, error) {
	fake.updatePolicyAssignmentMutex.Lock()
	ret, specificReturn := fake.updatePolicyAssignmentReturnsOnCall[len(fake.updatePolicyAssignmentArgsForCall)]
	fake.updatePolicyAssignmentArgsForCall = append(fake.updatePolicyAssignmentArgsForCall, struct {
		arg1 context.Context
		arg2 *v1alpha1.PolicyAssignment
	}{arg1, arg2})
	stub := fake.UpdatePolicyAssignmentStub
	fakeReturns := fake.updatePolicyAssignmentReturns
	fake.recordInvocation("UpdatePolicyAssignment", []interface{}{arg1, arg2})
	fake.updatePolicyAssignmentMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeAssignmentManager) UpdatePolicyAssignmentCallCount() int {
	fake.updatePolicyAssignmentMutex.RLock()
	defer fake.updatePolicyAssignmentMutex.RUnlock()
	return len(fake.updatePolicyAssignmentArgsForCall)
}

func (fake *FakeAssignmentManager) UpdatePolicyAssignmentCalls(stub func(context.Context, *v1alpha1.PolicyAssignment) (*v1alpha1.PolicyAssignment, error)) {
	fake.updatePolicyAssignmentMutex.Lock()
	defer fake.updatePolicyAssignmentMutex.Unlock()
	fake.UpdatePolicyAssignmentStub = stub
}

func (fake *FakeAssignmentManager) UpdatePolicyAssignmentArgsForCall(i int) (context.Context, *v1alpha1.PolicyAssignment) {
	fake.updatePolicyAssignmentMutex.RLock()
	defer fake.updatePolicyAssignmentMutex.RUnlock()
	argsForCall := fake.updatePolicyAssignmentArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeAssignmentManager) UpdatePolicyAssignmentReturns(result1 *v1alpha1.PolicyAssignment, result2 error) {
	fake.updatePolicyAssignmentMutex.Lock()
	defer fake.updatePolicyAssignmentMutex.Unlock()
	fake.UpdatePolicyAssignmentStub = nil
	fake.updatePolicyAssignmentReturns = struct {
		result1 *v1alpha1.PolicyAssignment
		result2 error
	}{result1, result2}
}

func (fake *FakeAssignmentManager) UpdatePolicyAssignmentReturnsOnCall(i int, result1 *v1alpha1.PolicyAssignment, result2 error) {
	fake.updatePolicyAssignmentMutex.Lock()
	defer fake.updatePolicyAssignmentMutex.Unlock()
	fake.UpdatePolicyAssignmentStub = nil
	if fake.updatePolicyAssignmentReturnsOnCall == nil {
		fake.updatePolicyAssignmentReturnsOnCall = make(map[int]struct {
			result1 *v1alpha1.PolicyAssignment
			result2 error
		})
	}
	fake.updatePolicyAssignmentReturnsOnCall[i] = struct {
		result1 *v1alpha1.PolicyAssignment
		result2 error
	}{result1, result2}
}

func (fake *FakeAssignmentManager) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.createPolicyAssignmentMutex.RLock()
	defer fake.createPolicyAssignmentMutex.RUnlock()
	fake.deletePolicyAssignmentMutex.RLock()
	defer fake.deletePolicyAssignmentMutex.RUnlock()
	fake.getPolicyAssignmentMutex.RLock()
	defer fake.getPolicyAssignmentMutex.RUnlock()
	fake.listPolicyAssignmentsMutex.RLock()
	defer fake.listPolicyAssignmentsMutex.RUnlock()
	fake.updatePolicyAssignmentMutex.RLock()
	defer fake.updatePolicyAssignmentMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeAssignmentManager) recordInvocation(key string, args []interface{}) {
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

var _ policy.AssignmentManager = new(FakeAssignmentManager)
