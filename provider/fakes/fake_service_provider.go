// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"context"
	"sync"

	"github.com/alphagov/paas-aiven-broker/provider"
	"github.com/pivotal-cf/brokerapi/domain"
)

type FakeServiceProvider struct {
	BindStub        func(context.Context, provider.BindData) (domain.Binding, error)
	bindMutex       sync.RWMutex
	bindArgsForCall []struct {
		arg1 context.Context
		arg2 provider.BindData
	}
	bindReturns struct {
		result1 domain.Binding
		result2 error
	}
	bindReturnsOnCall map[int]struct {
		result1 domain.Binding
		result2 error
	}
	DeprovisionStub        func(context.Context, provider.DeprovisionData) (string, error)
	deprovisionMutex       sync.RWMutex
	deprovisionArgsForCall []struct {
		arg1 context.Context
		arg2 provider.DeprovisionData
	}
	deprovisionReturns struct {
		result1 string
		result2 error
	}
	deprovisionReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	LastOperationStub        func(context.Context, provider.LastOperationData) (domain.LastOperationState, string, error)
	lastOperationMutex       sync.RWMutex
	lastOperationArgsForCall []struct {
		arg1 context.Context
		arg2 provider.LastOperationData
	}
	lastOperationReturns struct {
		result1 domain.LastOperationState
		result2 string
		result3 error
	}
	lastOperationReturnsOnCall map[int]struct {
		result1 domain.LastOperationState
		result2 string
		result3 error
	}
	ProvisionStub        func(context.Context, provider.ProvisionData, domain.ProvisionDetails) (string, string, error)
	provisionMutex       sync.RWMutex
	provisionArgsForCall []struct {
		arg1 context.Context
		arg2 provider.ProvisionData
		arg3 domain.ProvisionDetails
	}
	provisionReturns struct {
		result1 string
		result2 string
		result3 error
	}
	provisionReturnsOnCall map[int]struct {
		result1 string
		result2 string
		result3 error
	}
	UnbindStub        func(context.Context, provider.UnbindData) error
	unbindMutex       sync.RWMutex
	unbindArgsForCall []struct {
		arg1 context.Context
		arg2 provider.UnbindData
	}
	unbindReturns struct {
		result1 error
	}
	unbindReturnsOnCall map[int]struct {
		result1 error
	}
	UpdateStub        func(context.Context, provider.UpdateData, domain.UpdateDetails) (string, error)
	updateMutex       sync.RWMutex
	updateArgsForCall []struct {
		arg1 context.Context
		arg2 provider.UpdateData
		arg3 domain.UpdateDetails
	}
	updateReturns struct {
		result1 string
		result2 error
	}
	updateReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeServiceProvider) Bind(arg1 context.Context, arg2 provider.BindData) (domain.Binding, error) {
	fake.bindMutex.Lock()
	ret, specificReturn := fake.bindReturnsOnCall[len(fake.bindArgsForCall)]
	fake.bindArgsForCall = append(fake.bindArgsForCall, struct {
		arg1 context.Context
		arg2 provider.BindData
	}{arg1, arg2})
	fake.recordInvocation("Bind", []interface{}{arg1, arg2})
	fake.bindMutex.Unlock()
	if fake.BindStub != nil {
		return fake.BindStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.bindReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeServiceProvider) BindCallCount() int {
	fake.bindMutex.RLock()
	defer fake.bindMutex.RUnlock()
	return len(fake.bindArgsForCall)
}

func (fake *FakeServiceProvider) BindCalls(stub func(context.Context, provider.BindData) (domain.Binding, error)) {
	fake.bindMutex.Lock()
	defer fake.bindMutex.Unlock()
	fake.BindStub = stub
}

func (fake *FakeServiceProvider) BindArgsForCall(i int) (context.Context, provider.BindData) {
	fake.bindMutex.RLock()
	defer fake.bindMutex.RUnlock()
	argsForCall := fake.bindArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeServiceProvider) BindReturns(result1 domain.Binding, result2 error) {
	fake.bindMutex.Lock()
	defer fake.bindMutex.Unlock()
	fake.BindStub = nil
	fake.bindReturns = struct {
		result1 domain.Binding
		result2 error
	}{result1, result2}
}

func (fake *FakeServiceProvider) BindReturnsOnCall(i int, result1 domain.Binding, result2 error) {
	fake.bindMutex.Lock()
	defer fake.bindMutex.Unlock()
	fake.BindStub = nil
	if fake.bindReturnsOnCall == nil {
		fake.bindReturnsOnCall = make(map[int]struct {
			result1 domain.Binding
			result2 error
		})
	}
	fake.bindReturnsOnCall[i] = struct {
		result1 domain.Binding
		result2 error
	}{result1, result2}
}

func (fake *FakeServiceProvider) Deprovision(arg1 context.Context, arg2 provider.DeprovisionData) (string, error) {
	fake.deprovisionMutex.Lock()
	ret, specificReturn := fake.deprovisionReturnsOnCall[len(fake.deprovisionArgsForCall)]
	fake.deprovisionArgsForCall = append(fake.deprovisionArgsForCall, struct {
		arg1 context.Context
		arg2 provider.DeprovisionData
	}{arg1, arg2})
	fake.recordInvocation("Deprovision", []interface{}{arg1, arg2})
	fake.deprovisionMutex.Unlock()
	if fake.DeprovisionStub != nil {
		return fake.DeprovisionStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.deprovisionReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeServiceProvider) DeprovisionCallCount() int {
	fake.deprovisionMutex.RLock()
	defer fake.deprovisionMutex.RUnlock()
	return len(fake.deprovisionArgsForCall)
}

func (fake *FakeServiceProvider) DeprovisionCalls(stub func(context.Context, provider.DeprovisionData) (string, error)) {
	fake.deprovisionMutex.Lock()
	defer fake.deprovisionMutex.Unlock()
	fake.DeprovisionStub = stub
}

func (fake *FakeServiceProvider) DeprovisionArgsForCall(i int) (context.Context, provider.DeprovisionData) {
	fake.deprovisionMutex.RLock()
	defer fake.deprovisionMutex.RUnlock()
	argsForCall := fake.deprovisionArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeServiceProvider) DeprovisionReturns(result1 string, result2 error) {
	fake.deprovisionMutex.Lock()
	defer fake.deprovisionMutex.Unlock()
	fake.DeprovisionStub = nil
	fake.deprovisionReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeServiceProvider) DeprovisionReturnsOnCall(i int, result1 string, result2 error) {
	fake.deprovisionMutex.Lock()
	defer fake.deprovisionMutex.Unlock()
	fake.DeprovisionStub = nil
	if fake.deprovisionReturnsOnCall == nil {
		fake.deprovisionReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.deprovisionReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeServiceProvider) LastOperation(arg1 context.Context, arg2 provider.LastOperationData) (domain.LastOperationState, string, error) {
	fake.lastOperationMutex.Lock()
	ret, specificReturn := fake.lastOperationReturnsOnCall[len(fake.lastOperationArgsForCall)]
	fake.lastOperationArgsForCall = append(fake.lastOperationArgsForCall, struct {
		arg1 context.Context
		arg2 provider.LastOperationData
	}{arg1, arg2})
	fake.recordInvocation("LastOperation", []interface{}{arg1, arg2})
	fake.lastOperationMutex.Unlock()
	if fake.LastOperationStub != nil {
		return fake.LastOperationStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	fakeReturns := fake.lastOperationReturns
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *FakeServiceProvider) LastOperationCallCount() int {
	fake.lastOperationMutex.RLock()
	defer fake.lastOperationMutex.RUnlock()
	return len(fake.lastOperationArgsForCall)
}

func (fake *FakeServiceProvider) LastOperationCalls(stub func(context.Context, provider.LastOperationData) (domain.LastOperationState, string, error)) {
	fake.lastOperationMutex.Lock()
	defer fake.lastOperationMutex.Unlock()
	fake.LastOperationStub = stub
}

func (fake *FakeServiceProvider) LastOperationArgsForCall(i int) (context.Context, provider.LastOperationData) {
	fake.lastOperationMutex.RLock()
	defer fake.lastOperationMutex.RUnlock()
	argsForCall := fake.lastOperationArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeServiceProvider) LastOperationReturns(result1 domain.LastOperationState, result2 string, result3 error) {
	fake.lastOperationMutex.Lock()
	defer fake.lastOperationMutex.Unlock()
	fake.LastOperationStub = nil
	fake.lastOperationReturns = struct {
		result1 domain.LastOperationState
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeServiceProvider) LastOperationReturnsOnCall(i int, result1 domain.LastOperationState, result2 string, result3 error) {
	fake.lastOperationMutex.Lock()
	defer fake.lastOperationMutex.Unlock()
	fake.LastOperationStub = nil
	if fake.lastOperationReturnsOnCall == nil {
		fake.lastOperationReturnsOnCall = make(map[int]struct {
			result1 domain.LastOperationState
			result2 string
			result3 error
		})
	}
	fake.lastOperationReturnsOnCall[i] = struct {
		result1 domain.LastOperationState
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeServiceProvider) Provision(arg1 context.Context, arg2 provider.ProvisionData, arg3 domain.ProvisionDetails) (string, string, error) {
	fake.provisionMutex.Lock()
	ret, specificReturn := fake.provisionReturnsOnCall[len(fake.provisionArgsForCall)]
	fake.provisionArgsForCall = append(fake.provisionArgsForCall, struct {
		arg1 context.Context
		arg2 provider.ProvisionData
		arg3 domain.ProvisionDetails
	}{arg1, arg2, arg3})
	fake.recordInvocation("Provision", []interface{}{arg1, arg2, arg3})
	fake.provisionMutex.Unlock()
	if fake.ProvisionStub != nil {
		return fake.ProvisionStub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	fakeReturns := fake.provisionReturns
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *FakeServiceProvider) ProvisionCallCount() int {
	fake.provisionMutex.RLock()
	defer fake.provisionMutex.RUnlock()
	return len(fake.provisionArgsForCall)
}

func (fake *FakeServiceProvider) ProvisionCalls(stub func(context.Context, provider.ProvisionData, domain.ProvisionDetails) (string, string, error)) {
	fake.provisionMutex.Lock()
	defer fake.provisionMutex.Unlock()
	fake.ProvisionStub = stub
}

func (fake *FakeServiceProvider) ProvisionArgsForCall(i int) (context.Context, provider.ProvisionData, domain.ProvisionDetails) {
	fake.provisionMutex.RLock()
	defer fake.provisionMutex.RUnlock()
	argsForCall := fake.provisionArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeServiceProvider) ProvisionReturns(result1 string, result2 string, result3 error) {
	fake.provisionMutex.Lock()
	defer fake.provisionMutex.Unlock()
	fake.ProvisionStub = nil
	fake.provisionReturns = struct {
		result1 string
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeServiceProvider) ProvisionReturnsOnCall(i int, result1 string, result2 string, result3 error) {
	fake.provisionMutex.Lock()
	defer fake.provisionMutex.Unlock()
	fake.ProvisionStub = nil
	if fake.provisionReturnsOnCall == nil {
		fake.provisionReturnsOnCall = make(map[int]struct {
			result1 string
			result2 string
			result3 error
		})
	}
	fake.provisionReturnsOnCall[i] = struct {
		result1 string
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeServiceProvider) Unbind(arg1 context.Context, arg2 provider.UnbindData) error {
	fake.unbindMutex.Lock()
	ret, specificReturn := fake.unbindReturnsOnCall[len(fake.unbindArgsForCall)]
	fake.unbindArgsForCall = append(fake.unbindArgsForCall, struct {
		arg1 context.Context
		arg2 provider.UnbindData
	}{arg1, arg2})
	fake.recordInvocation("Unbind", []interface{}{arg1, arg2})
	fake.unbindMutex.Unlock()
	if fake.UnbindStub != nil {
		return fake.UnbindStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.unbindReturns
	return fakeReturns.result1
}

func (fake *FakeServiceProvider) UnbindCallCount() int {
	fake.unbindMutex.RLock()
	defer fake.unbindMutex.RUnlock()
	return len(fake.unbindArgsForCall)
}

func (fake *FakeServiceProvider) UnbindCalls(stub func(context.Context, provider.UnbindData) error) {
	fake.unbindMutex.Lock()
	defer fake.unbindMutex.Unlock()
	fake.UnbindStub = stub
}

func (fake *FakeServiceProvider) UnbindArgsForCall(i int) (context.Context, provider.UnbindData) {
	fake.unbindMutex.RLock()
	defer fake.unbindMutex.RUnlock()
	argsForCall := fake.unbindArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeServiceProvider) UnbindReturns(result1 error) {
	fake.unbindMutex.Lock()
	defer fake.unbindMutex.Unlock()
	fake.UnbindStub = nil
	fake.unbindReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeServiceProvider) UnbindReturnsOnCall(i int, result1 error) {
	fake.unbindMutex.Lock()
	defer fake.unbindMutex.Unlock()
	fake.UnbindStub = nil
	if fake.unbindReturnsOnCall == nil {
		fake.unbindReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.unbindReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeServiceProvider) Update(arg1 context.Context, arg2 provider.UpdateData, arg3 domain.UpdateDetails) (string, error) {
	fake.updateMutex.Lock()
	ret, specificReturn := fake.updateReturnsOnCall[len(fake.updateArgsForCall)]
	fake.updateArgsForCall = append(fake.updateArgsForCall, struct {
		arg1 context.Context
		arg2 provider.UpdateData
		arg3 domain.UpdateDetails
	}{arg1, arg2, arg3})
	fake.recordInvocation("Update", []interface{}{arg1, arg2, arg3})
	fake.updateMutex.Unlock()
	if fake.UpdateStub != nil {
		return fake.UpdateStub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.updateReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeServiceProvider) UpdateCallCount() int {
	fake.updateMutex.RLock()
	defer fake.updateMutex.RUnlock()
	return len(fake.updateArgsForCall)
}

func (fake *FakeServiceProvider) UpdateCalls(stub func(context.Context, provider.UpdateData, domain.UpdateDetails) (string, error)) {
	fake.updateMutex.Lock()
	defer fake.updateMutex.Unlock()
	fake.UpdateStub = stub
}

func (fake *FakeServiceProvider) UpdateArgsForCall(i int) (context.Context, provider.UpdateData, domain.UpdateDetails) {
	fake.updateMutex.RLock()
	defer fake.updateMutex.RUnlock()
	argsForCall := fake.updateArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeServiceProvider) UpdateReturns(result1 string, result2 error) {
	fake.updateMutex.Lock()
	defer fake.updateMutex.Unlock()
	fake.UpdateStub = nil
	fake.updateReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeServiceProvider) UpdateReturnsOnCall(i int, result1 string, result2 error) {
	fake.updateMutex.Lock()
	defer fake.updateMutex.Unlock()
	fake.UpdateStub = nil
	if fake.updateReturnsOnCall == nil {
		fake.updateReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.updateReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeServiceProvider) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.bindMutex.RLock()
	defer fake.bindMutex.RUnlock()
	fake.deprovisionMutex.RLock()
	defer fake.deprovisionMutex.RUnlock()
	fake.lastOperationMutex.RLock()
	defer fake.lastOperationMutex.RUnlock()
	fake.provisionMutex.RLock()
	defer fake.provisionMutex.RUnlock()
	fake.unbindMutex.RLock()
	defer fake.unbindMutex.RUnlock()
	fake.updateMutex.RLock()
	defer fake.updateMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeServiceProvider) recordInvocation(key string, args []interface{}) {
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

var _ provider.ServiceProvider = new(FakeServiceProvider)
