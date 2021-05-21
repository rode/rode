// Code generated by counterfeiter. DO NOT EDIT.
package resourcefakes

import (
	"context"
	"sync"

	"github.com/rode/rode/pkg/resource"
	"github.com/rode/rode/proto/v1alpha1"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
)

type FakeManager struct {
	BatchCreateGenericResourceVersionsStub        func(context.Context, []*grafeas_go_proto.Occurrence) error
	batchCreateGenericResourceVersionsMutex       sync.RWMutex
	batchCreateGenericResourceVersionsArgsForCall []struct {
		arg1 context.Context
		arg2 []*grafeas_go_proto.Occurrence
	}
	batchCreateGenericResourceVersionsReturns struct {
		result1 error
	}
	batchCreateGenericResourceVersionsReturnsOnCall map[int]struct {
		result1 error
	}
	BatchCreateGenericResourcesStub        func(context.Context, []*grafeas_go_proto.Occurrence) error
	batchCreateGenericResourcesMutex       sync.RWMutex
	batchCreateGenericResourcesArgsForCall []struct {
		arg1 context.Context
		arg2 []*grafeas_go_proto.Occurrence
	}
	batchCreateGenericResourcesReturns struct {
		result1 error
	}
	batchCreateGenericResourcesReturnsOnCall map[int]struct {
		result1 error
	}
	GetGenericResourceStub        func(context.Context, string) (*v1alpha1.GenericResource, error)
	getGenericResourceMutex       sync.RWMutex
	getGenericResourceArgsForCall []struct {
		arg1 context.Context
		arg2 string
	}
	getGenericResourceReturns struct {
		result1 *v1alpha1.GenericResource
		result2 error
	}
	getGenericResourceReturnsOnCall map[int]struct {
		result1 *v1alpha1.GenericResource
		result2 error
	}
	ListGenericResourceVersionsStub        func(context.Context, *v1alpha1.ListGenericResourceVersionsRequest) (*v1alpha1.ListGenericResourceVersionsResponse, error)
	listGenericResourceVersionsMutex       sync.RWMutex
	listGenericResourceVersionsArgsForCall []struct {
		arg1 context.Context
		arg2 *v1alpha1.ListGenericResourceVersionsRequest
	}
	listGenericResourceVersionsReturns struct {
		result1 *v1alpha1.ListGenericResourceVersionsResponse
		result2 error
	}
	listGenericResourceVersionsReturnsOnCall map[int]struct {
		result1 *v1alpha1.ListGenericResourceVersionsResponse
		result2 error
	}
	ListGenericResourcesStub        func(context.Context, *v1alpha1.ListGenericResourcesRequest) (*v1alpha1.ListGenericResourcesResponse, error)
	listGenericResourcesMutex       sync.RWMutex
	listGenericResourcesArgsForCall []struct {
		arg1 context.Context
		arg2 *v1alpha1.ListGenericResourcesRequest
	}
	listGenericResourcesReturns struct {
		result1 *v1alpha1.ListGenericResourcesResponse
		result2 error
	}
	listGenericResourcesReturnsOnCall map[int]struct {
		result1 *v1alpha1.ListGenericResourcesResponse
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeManager) BatchCreateGenericResourceVersions(arg1 context.Context, arg2 []*grafeas_go_proto.Occurrence) error {
	var arg2Copy []*grafeas_go_proto.Occurrence
	if arg2 != nil {
		arg2Copy = make([]*grafeas_go_proto.Occurrence, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.batchCreateGenericResourceVersionsMutex.Lock()
	ret, specificReturn := fake.batchCreateGenericResourceVersionsReturnsOnCall[len(fake.batchCreateGenericResourceVersionsArgsForCall)]
	fake.batchCreateGenericResourceVersionsArgsForCall = append(fake.batchCreateGenericResourceVersionsArgsForCall, struct {
		arg1 context.Context
		arg2 []*grafeas_go_proto.Occurrence
	}{arg1, arg2Copy})
	stub := fake.BatchCreateGenericResourceVersionsStub
	fakeReturns := fake.batchCreateGenericResourceVersionsReturns
	fake.recordInvocation("BatchCreateGenericResourceVersions", []interface{}{arg1, arg2Copy})
	fake.batchCreateGenericResourceVersionsMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeManager) BatchCreateGenericResourceVersionsCallCount() int {
	fake.batchCreateGenericResourceVersionsMutex.RLock()
	defer fake.batchCreateGenericResourceVersionsMutex.RUnlock()
	return len(fake.batchCreateGenericResourceVersionsArgsForCall)
}

func (fake *FakeManager) BatchCreateGenericResourceVersionsCalls(stub func(context.Context, []*grafeas_go_proto.Occurrence) error) {
	fake.batchCreateGenericResourceVersionsMutex.Lock()
	defer fake.batchCreateGenericResourceVersionsMutex.Unlock()
	fake.BatchCreateGenericResourceVersionsStub = stub
}

func (fake *FakeManager) BatchCreateGenericResourceVersionsArgsForCall(i int) (context.Context, []*grafeas_go_proto.Occurrence) {
	fake.batchCreateGenericResourceVersionsMutex.RLock()
	defer fake.batchCreateGenericResourceVersionsMutex.RUnlock()
	argsForCall := fake.batchCreateGenericResourceVersionsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeManager) BatchCreateGenericResourceVersionsReturns(result1 error) {
	fake.batchCreateGenericResourceVersionsMutex.Lock()
	defer fake.batchCreateGenericResourceVersionsMutex.Unlock()
	fake.BatchCreateGenericResourceVersionsStub = nil
	fake.batchCreateGenericResourceVersionsReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeManager) BatchCreateGenericResourceVersionsReturnsOnCall(i int, result1 error) {
	fake.batchCreateGenericResourceVersionsMutex.Lock()
	defer fake.batchCreateGenericResourceVersionsMutex.Unlock()
	fake.BatchCreateGenericResourceVersionsStub = nil
	if fake.batchCreateGenericResourceVersionsReturnsOnCall == nil {
		fake.batchCreateGenericResourceVersionsReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.batchCreateGenericResourceVersionsReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeManager) BatchCreateGenericResources(arg1 context.Context, arg2 []*grafeas_go_proto.Occurrence) error {
	var arg2Copy []*grafeas_go_proto.Occurrence
	if arg2 != nil {
		arg2Copy = make([]*grafeas_go_proto.Occurrence, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.batchCreateGenericResourcesMutex.Lock()
	ret, specificReturn := fake.batchCreateGenericResourcesReturnsOnCall[len(fake.batchCreateGenericResourcesArgsForCall)]
	fake.batchCreateGenericResourcesArgsForCall = append(fake.batchCreateGenericResourcesArgsForCall, struct {
		arg1 context.Context
		arg2 []*grafeas_go_proto.Occurrence
	}{arg1, arg2Copy})
	stub := fake.BatchCreateGenericResourcesStub
	fakeReturns := fake.batchCreateGenericResourcesReturns
	fake.recordInvocation("BatchCreateGenericResources", []interface{}{arg1, arg2Copy})
	fake.batchCreateGenericResourcesMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeManager) BatchCreateGenericResourcesCallCount() int {
	fake.batchCreateGenericResourcesMutex.RLock()
	defer fake.batchCreateGenericResourcesMutex.RUnlock()
	return len(fake.batchCreateGenericResourcesArgsForCall)
}

func (fake *FakeManager) BatchCreateGenericResourcesCalls(stub func(context.Context, []*grafeas_go_proto.Occurrence) error) {
	fake.batchCreateGenericResourcesMutex.Lock()
	defer fake.batchCreateGenericResourcesMutex.Unlock()
	fake.BatchCreateGenericResourcesStub = stub
}

func (fake *FakeManager) BatchCreateGenericResourcesArgsForCall(i int) (context.Context, []*grafeas_go_proto.Occurrence) {
	fake.batchCreateGenericResourcesMutex.RLock()
	defer fake.batchCreateGenericResourcesMutex.RUnlock()
	argsForCall := fake.batchCreateGenericResourcesArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeManager) BatchCreateGenericResourcesReturns(result1 error) {
	fake.batchCreateGenericResourcesMutex.Lock()
	defer fake.batchCreateGenericResourcesMutex.Unlock()
	fake.BatchCreateGenericResourcesStub = nil
	fake.batchCreateGenericResourcesReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeManager) BatchCreateGenericResourcesReturnsOnCall(i int, result1 error) {
	fake.batchCreateGenericResourcesMutex.Lock()
	defer fake.batchCreateGenericResourcesMutex.Unlock()
	fake.BatchCreateGenericResourcesStub = nil
	if fake.batchCreateGenericResourcesReturnsOnCall == nil {
		fake.batchCreateGenericResourcesReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.batchCreateGenericResourcesReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeManager) GetGenericResource(arg1 context.Context, arg2 string) (*v1alpha1.GenericResource, error) {
	fake.getGenericResourceMutex.Lock()
	ret, specificReturn := fake.getGenericResourceReturnsOnCall[len(fake.getGenericResourceArgsForCall)]
	fake.getGenericResourceArgsForCall = append(fake.getGenericResourceArgsForCall, struct {
		arg1 context.Context
		arg2 string
	}{arg1, arg2})
	stub := fake.GetGenericResourceStub
	fakeReturns := fake.getGenericResourceReturns
	fake.recordInvocation("GetGenericResource", []interface{}{arg1, arg2})
	fake.getGenericResourceMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeManager) GetGenericResourceCallCount() int {
	fake.getGenericResourceMutex.RLock()
	defer fake.getGenericResourceMutex.RUnlock()
	return len(fake.getGenericResourceArgsForCall)
}

func (fake *FakeManager) GetGenericResourceCalls(stub func(context.Context, string) (*v1alpha1.GenericResource, error)) {
	fake.getGenericResourceMutex.Lock()
	defer fake.getGenericResourceMutex.Unlock()
	fake.GetGenericResourceStub = stub
}

func (fake *FakeManager) GetGenericResourceArgsForCall(i int) (context.Context, string) {
	fake.getGenericResourceMutex.RLock()
	defer fake.getGenericResourceMutex.RUnlock()
	argsForCall := fake.getGenericResourceArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeManager) GetGenericResourceReturns(result1 *v1alpha1.GenericResource, result2 error) {
	fake.getGenericResourceMutex.Lock()
	defer fake.getGenericResourceMutex.Unlock()
	fake.GetGenericResourceStub = nil
	fake.getGenericResourceReturns = struct {
		result1 *v1alpha1.GenericResource
		result2 error
	}{result1, result2}
}

func (fake *FakeManager) GetGenericResourceReturnsOnCall(i int, result1 *v1alpha1.GenericResource, result2 error) {
	fake.getGenericResourceMutex.Lock()
	defer fake.getGenericResourceMutex.Unlock()
	fake.GetGenericResourceStub = nil
	if fake.getGenericResourceReturnsOnCall == nil {
		fake.getGenericResourceReturnsOnCall = make(map[int]struct {
			result1 *v1alpha1.GenericResource
			result2 error
		})
	}
	fake.getGenericResourceReturnsOnCall[i] = struct {
		result1 *v1alpha1.GenericResource
		result2 error
	}{result1, result2}
}

func (fake *FakeManager) ListGenericResourceVersions(arg1 context.Context, arg2 *v1alpha1.ListGenericResourceVersionsRequest) (*v1alpha1.ListGenericResourceVersionsResponse, error) {
	fake.listGenericResourceVersionsMutex.Lock()
	ret, specificReturn := fake.listGenericResourceVersionsReturnsOnCall[len(fake.listGenericResourceVersionsArgsForCall)]
	fake.listGenericResourceVersionsArgsForCall = append(fake.listGenericResourceVersionsArgsForCall, struct {
		arg1 context.Context
		arg2 *v1alpha1.ListGenericResourceVersionsRequest
	}{arg1, arg2})
	stub := fake.ListGenericResourceVersionsStub
	fakeReturns := fake.listGenericResourceVersionsReturns
	fake.recordInvocation("ListGenericResourceVersions", []interface{}{arg1, arg2})
	fake.listGenericResourceVersionsMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeManager) ListGenericResourceVersionsCallCount() int {
	fake.listGenericResourceVersionsMutex.RLock()
	defer fake.listGenericResourceVersionsMutex.RUnlock()
	return len(fake.listGenericResourceVersionsArgsForCall)
}

func (fake *FakeManager) ListGenericResourceVersionsCalls(stub func(context.Context, *v1alpha1.ListGenericResourceVersionsRequest) (*v1alpha1.ListGenericResourceVersionsResponse, error)) {
	fake.listGenericResourceVersionsMutex.Lock()
	defer fake.listGenericResourceVersionsMutex.Unlock()
	fake.ListGenericResourceVersionsStub = stub
}

func (fake *FakeManager) ListGenericResourceVersionsArgsForCall(i int) (context.Context, *v1alpha1.ListGenericResourceVersionsRequest) {
	fake.listGenericResourceVersionsMutex.RLock()
	defer fake.listGenericResourceVersionsMutex.RUnlock()
	argsForCall := fake.listGenericResourceVersionsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeManager) ListGenericResourceVersionsReturns(result1 *v1alpha1.ListGenericResourceVersionsResponse, result2 error) {
	fake.listGenericResourceVersionsMutex.Lock()
	defer fake.listGenericResourceVersionsMutex.Unlock()
	fake.ListGenericResourceVersionsStub = nil
	fake.listGenericResourceVersionsReturns = struct {
		result1 *v1alpha1.ListGenericResourceVersionsResponse
		result2 error
	}{result1, result2}
}

func (fake *FakeManager) ListGenericResourceVersionsReturnsOnCall(i int, result1 *v1alpha1.ListGenericResourceVersionsResponse, result2 error) {
	fake.listGenericResourceVersionsMutex.Lock()
	defer fake.listGenericResourceVersionsMutex.Unlock()
	fake.ListGenericResourceVersionsStub = nil
	if fake.listGenericResourceVersionsReturnsOnCall == nil {
		fake.listGenericResourceVersionsReturnsOnCall = make(map[int]struct {
			result1 *v1alpha1.ListGenericResourceVersionsResponse
			result2 error
		})
	}
	fake.listGenericResourceVersionsReturnsOnCall[i] = struct {
		result1 *v1alpha1.ListGenericResourceVersionsResponse
		result2 error
	}{result1, result2}
}

func (fake *FakeManager) ListGenericResources(arg1 context.Context, arg2 *v1alpha1.ListGenericResourcesRequest) (*v1alpha1.ListGenericResourcesResponse, error) {
	fake.listGenericResourcesMutex.Lock()
	ret, specificReturn := fake.listGenericResourcesReturnsOnCall[len(fake.listGenericResourcesArgsForCall)]
	fake.listGenericResourcesArgsForCall = append(fake.listGenericResourcesArgsForCall, struct {
		arg1 context.Context
		arg2 *v1alpha1.ListGenericResourcesRequest
	}{arg1, arg2})
	stub := fake.ListGenericResourcesStub
	fakeReturns := fake.listGenericResourcesReturns
	fake.recordInvocation("ListGenericResources", []interface{}{arg1, arg2})
	fake.listGenericResourcesMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeManager) ListGenericResourcesCallCount() int {
	fake.listGenericResourcesMutex.RLock()
	defer fake.listGenericResourcesMutex.RUnlock()
	return len(fake.listGenericResourcesArgsForCall)
}

func (fake *FakeManager) ListGenericResourcesCalls(stub func(context.Context, *v1alpha1.ListGenericResourcesRequest) (*v1alpha1.ListGenericResourcesResponse, error)) {
	fake.listGenericResourcesMutex.Lock()
	defer fake.listGenericResourcesMutex.Unlock()
	fake.ListGenericResourcesStub = stub
}

func (fake *FakeManager) ListGenericResourcesArgsForCall(i int) (context.Context, *v1alpha1.ListGenericResourcesRequest) {
	fake.listGenericResourcesMutex.RLock()
	defer fake.listGenericResourcesMutex.RUnlock()
	argsForCall := fake.listGenericResourcesArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeManager) ListGenericResourcesReturns(result1 *v1alpha1.ListGenericResourcesResponse, result2 error) {
	fake.listGenericResourcesMutex.Lock()
	defer fake.listGenericResourcesMutex.Unlock()
	fake.ListGenericResourcesStub = nil
	fake.listGenericResourcesReturns = struct {
		result1 *v1alpha1.ListGenericResourcesResponse
		result2 error
	}{result1, result2}
}

func (fake *FakeManager) ListGenericResourcesReturnsOnCall(i int, result1 *v1alpha1.ListGenericResourcesResponse, result2 error) {
	fake.listGenericResourcesMutex.Lock()
	defer fake.listGenericResourcesMutex.Unlock()
	fake.ListGenericResourcesStub = nil
	if fake.listGenericResourcesReturnsOnCall == nil {
		fake.listGenericResourcesReturnsOnCall = make(map[int]struct {
			result1 *v1alpha1.ListGenericResourcesResponse
			result2 error
		})
	}
	fake.listGenericResourcesReturnsOnCall[i] = struct {
		result1 *v1alpha1.ListGenericResourcesResponse
		result2 error
	}{result1, result2}
}

func (fake *FakeManager) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.batchCreateGenericResourceVersionsMutex.RLock()
	defer fake.batchCreateGenericResourceVersionsMutex.RUnlock()
	fake.batchCreateGenericResourcesMutex.RLock()
	defer fake.batchCreateGenericResourcesMutex.RUnlock()
	fake.getGenericResourceMutex.RLock()
	defer fake.getGenericResourceMutex.RUnlock()
	fake.listGenericResourceVersionsMutex.RLock()
	defer fake.listGenericResourceVersionsMutex.RUnlock()
	fake.listGenericResourcesMutex.RLock()
	defer fake.listGenericResourcesMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeManager) recordInvocation(key string, args []interface{}) {
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

var _ resource.Manager = new(FakeManager)
