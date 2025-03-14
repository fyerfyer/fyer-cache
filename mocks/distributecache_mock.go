// Code generated by mockery v2.50.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	time "time"
)

// DistributedCache is an autogenerated mock type for the DistributedCache type
type DistributedCache struct {
	mock.Mock
}

type DistributedCache_Expecter struct {
	mock *mock.Mock
}

func (_m *DistributedCache) EXPECT() *DistributedCache_Expecter {
	return &DistributedCache_Expecter{mock: &_m.Mock}
}

// Del provides a mock function with given fields: ctx, key
func (_m *DistributedCache) Del(ctx context.Context, key string) error {
	ret := _m.Called(ctx, key)

	if len(ret) == 0 {
		panic("no return value specified for Del")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DistributedCache_Del_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Del'
type DistributedCache_Del_Call struct {
	*mock.Call
}

// Del is a helper method to define mock.On call
//   - ctx context.Context
//   - key string
func (_e *DistributedCache_Expecter) Del(ctx interface{}, key interface{}) *DistributedCache_Del_Call {
	return &DistributedCache_Del_Call{Call: _e.mock.On("Del", ctx, key)}
}

func (_c *DistributedCache_Del_Call) Run(run func(ctx context.Context, key string)) *DistributedCache_Del_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *DistributedCache_Del_Call) Return(_a0 error) *DistributedCache_Del_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DistributedCache_Del_Call) RunAndReturn(run func(context.Context, string) error) *DistributedCache_Del_Call {
	_c.Call.Return(run)
	return _c
}

// DelLocal provides a mock function with given fields: ctx, key
func (_m *DistributedCache) DelLocal(ctx context.Context, key string) error {
	ret := _m.Called(ctx, key)

	if len(ret) == 0 {
		panic("no return value specified for DelLocal")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DistributedCache_DelLocal_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DelLocal'
type DistributedCache_DelLocal_Call struct {
	*mock.Call
}

// DelLocal is a helper method to define mock.On call
//   - ctx context.Context
//   - key string
func (_e *DistributedCache_Expecter) DelLocal(ctx interface{}, key interface{}) *DistributedCache_DelLocal_Call {
	return &DistributedCache_DelLocal_Call{Call: _e.mock.On("DelLocal", ctx, key)}
}

func (_c *DistributedCache_DelLocal_Call) Run(run func(ctx context.Context, key string)) *DistributedCache_DelLocal_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *DistributedCache_DelLocal_Call) Return(_a0 error) *DistributedCache_DelLocal_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DistributedCache_DelLocal_Call) RunAndReturn(run func(context.Context, string) error) *DistributedCache_DelLocal_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: ctx, key
func (_m *DistributedCache) Get(ctx context.Context, key string) (interface{}, error) {
	ret := _m.Called(ctx, key)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 interface{}
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (interface{}, error)); ok {
		return rf(ctx, key)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) interface{}); ok {
		r0 = rf(ctx, key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DistributedCache_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type DistributedCache_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
//   - key string
func (_e *DistributedCache_Expecter) Get(ctx interface{}, key interface{}) *DistributedCache_Get_Call {
	return &DistributedCache_Get_Call{Call: _e.mock.On("Get", ctx, key)}
}

func (_c *DistributedCache_Get_Call) Run(run func(ctx context.Context, key string)) *DistributedCache_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *DistributedCache_Get_Call) Return(_a0 interface{}, _a1 error) *DistributedCache_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DistributedCache_Get_Call) RunAndReturn(run func(context.Context, string) (interface{}, error)) *DistributedCache_Get_Call {
	_c.Call.Return(run)
	return _c
}

// GetLocal provides a mock function with given fields: ctx, key
func (_m *DistributedCache) GetLocal(ctx context.Context, key string) (interface{}, error) {
	ret := _m.Called(ctx, key)

	if len(ret) == 0 {
		panic("no return value specified for GetLocal")
	}

	var r0 interface{}
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (interface{}, error)); ok {
		return rf(ctx, key)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) interface{}); ok {
		r0 = rf(ctx, key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DistributedCache_GetLocal_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetLocal'
type DistributedCache_GetLocal_Call struct {
	*mock.Call
}

// GetLocal is a helper method to define mock.On call
//   - ctx context.Context
//   - key string
func (_e *DistributedCache_Expecter) GetLocal(ctx interface{}, key interface{}) *DistributedCache_GetLocal_Call {
	return &DistributedCache_GetLocal_Call{Call: _e.mock.On("GetLocal", ctx, key)}
}

func (_c *DistributedCache_GetLocal_Call) Run(run func(ctx context.Context, key string)) *DistributedCache_GetLocal_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *DistributedCache_GetLocal_Call) Return(_a0 interface{}, _a1 error) *DistributedCache_GetLocal_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DistributedCache_GetLocal_Call) RunAndReturn(run func(context.Context, string) (interface{}, error)) *DistributedCache_GetLocal_Call {
	_c.Call.Return(run)
	return _c
}

// NodeAddress provides a mock function with no fields
func (_m *DistributedCache) NodeAddress() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for NodeAddress")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// DistributedCache_NodeAddress_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'NodeAddress'
type DistributedCache_NodeAddress_Call struct {
	*mock.Call
}

// NodeAddress is a helper method to define mock.On call
func (_e *DistributedCache_Expecter) NodeAddress() *DistributedCache_NodeAddress_Call {
	return &DistributedCache_NodeAddress_Call{Call: _e.mock.On("NodeAddress")}
}

func (_c *DistributedCache_NodeAddress_Call) Run(run func()) *DistributedCache_NodeAddress_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *DistributedCache_NodeAddress_Call) Return(_a0 string) *DistributedCache_NodeAddress_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DistributedCache_NodeAddress_Call) RunAndReturn(run func() string) *DistributedCache_NodeAddress_Call {
	_c.Call.Return(run)
	return _c
}

// NodeID provides a mock function with no fields
func (_m *DistributedCache) NodeID() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for NodeID")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// DistributedCache_NodeID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'NodeID'
type DistributedCache_NodeID_Call struct {
	*mock.Call
}

// NodeID is a helper method to define mock.On call
func (_e *DistributedCache_Expecter) NodeID() *DistributedCache_NodeID_Call {
	return &DistributedCache_NodeID_Call{Call: _e.mock.On("NodeID")}
}

func (_c *DistributedCache_NodeID_Call) Run(run func()) *DistributedCache_NodeID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *DistributedCache_NodeID_Call) Return(_a0 string) *DistributedCache_NodeID_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DistributedCache_NodeID_Call) RunAndReturn(run func() string) *DistributedCache_NodeID_Call {
	_c.Call.Return(run)
	return _c
}

// Set provides a mock function with given fields: ctx, key, value, expiration
func (_m *DistributedCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	ret := _m.Called(ctx, key, value, expiration)

	if len(ret) == 0 {
		panic("no return value specified for Set")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, interface{}, time.Duration) error); ok {
		r0 = rf(ctx, key, value, expiration)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DistributedCache_Set_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Set'
type DistributedCache_Set_Call struct {
	*mock.Call
}

// Set is a helper method to define mock.On call
//   - ctx context.Context
//   - key string
//   - value interface{}
//   - expiration time.Duration
func (_e *DistributedCache_Expecter) Set(ctx interface{}, key interface{}, value interface{}, expiration interface{}) *DistributedCache_Set_Call {
	return &DistributedCache_Set_Call{Call: _e.mock.On("Set", ctx, key, value, expiration)}
}

func (_c *DistributedCache_Set_Call) Run(run func(ctx context.Context, key string, value interface{}, expiration time.Duration)) *DistributedCache_Set_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(interface{}), args[3].(time.Duration))
	})
	return _c
}

func (_c *DistributedCache_Set_Call) Return(_a0 error) *DistributedCache_Set_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DistributedCache_Set_Call) RunAndReturn(run func(context.Context, string, interface{}, time.Duration) error) *DistributedCache_Set_Call {
	_c.Call.Return(run)
	return _c
}

// SetLocal provides a mock function with given fields: ctx, key, value, expiration
func (_m *DistributedCache) SetLocal(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	ret := _m.Called(ctx, key, value, expiration)

	if len(ret) == 0 {
		panic("no return value specified for SetLocal")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, interface{}, time.Duration) error); ok {
		r0 = rf(ctx, key, value, expiration)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DistributedCache_SetLocal_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetLocal'
type DistributedCache_SetLocal_Call struct {
	*mock.Call
}

// SetLocal is a helper method to define mock.On call
//   - ctx context.Context
//   - key string
//   - value interface{}
//   - expiration time.Duration
func (_e *DistributedCache_Expecter) SetLocal(ctx interface{}, key interface{}, value interface{}, expiration interface{}) *DistributedCache_SetLocal_Call {
	return &DistributedCache_SetLocal_Call{Call: _e.mock.On("SetLocal", ctx, key, value, expiration)}
}

func (_c *DistributedCache_SetLocal_Call) Run(run func(ctx context.Context, key string, value interface{}, expiration time.Duration)) *DistributedCache_SetLocal_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(interface{}), args[3].(time.Duration))
	})
	return _c
}

func (_c *DistributedCache_SetLocal_Call) Return(_a0 error) *DistributedCache_SetLocal_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DistributedCache_SetLocal_Call) RunAndReturn(run func(context.Context, string, interface{}, time.Duration) error) *DistributedCache_SetLocal_Call {
	_c.Call.Return(run)
	return _c
}

// NewDistributedCache creates a new instance of DistributedCache. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDistributedCache(t interface {
	mock.TestingT
	Cleanup(func())
}) *DistributedCache {
	mock := &DistributedCache{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
