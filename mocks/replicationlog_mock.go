// Code generated by mockery v2.50.0. DO NOT EDIT.

package mocks

import (
	replication "github.com/fyerfyer/fyer-cache/cache/replication"
	mock "github.com/stretchr/testify/mock"
)

// ReplicationLog is an autogenerated mock type for the ReplicationLog type
type ReplicationLog struct {
	mock.Mock
}

type ReplicationLog_Expecter struct {
	mock *mock.Mock
}

func (_m *ReplicationLog) EXPECT() *ReplicationLog_Expecter {
	return &ReplicationLog_Expecter{mock: &_m.Mock}
}

// Append provides a mock function with given fields: entry
func (_m *ReplicationLog) Append(entry *replication.ReplicationEntry) error {
	ret := _m.Called(entry)

	if len(ret) == 0 {
		panic("no return value specified for Append")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*replication.ReplicationEntry) error); ok {
		r0 = rf(entry)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ReplicationLog_Append_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Append'
type ReplicationLog_Append_Call struct {
	*mock.Call
}

// Append is a helper method to define mock.On call
//   - entry *replication.ReplicationEntry
func (_e *ReplicationLog_Expecter) Append(entry interface{}) *ReplicationLog_Append_Call {
	return &ReplicationLog_Append_Call{Call: _e.mock.On("Append", entry)}
}

func (_c *ReplicationLog_Append_Call) Run(run func(entry *replication.ReplicationEntry)) *ReplicationLog_Append_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*replication.ReplicationEntry))
	})
	return _c
}

func (_c *ReplicationLog_Append_Call) Return(_a0 error) *ReplicationLog_Append_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ReplicationLog_Append_Call) RunAndReturn(run func(*replication.ReplicationEntry) error) *ReplicationLog_Append_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: index
func (_m *ReplicationLog) Get(index uint64) (*replication.ReplicationEntry, error) {
	ret := _m.Called(index)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 *replication.ReplicationEntry
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (*replication.ReplicationEntry, error)); ok {
		return rf(index)
	}
	if rf, ok := ret.Get(0).(func(uint64) *replication.ReplicationEntry); ok {
		r0 = rf(index)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*replication.ReplicationEntry)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(index)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ReplicationLog_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type ReplicationLog_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - index uint64
func (_e *ReplicationLog_Expecter) Get(index interface{}) *ReplicationLog_Get_Call {
	return &ReplicationLog_Get_Call{Call: _e.mock.On("Get", index)}
}

func (_c *ReplicationLog_Get_Call) Run(run func(index uint64)) *ReplicationLog_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *ReplicationLog_Get_Call) Return(_a0 *replication.ReplicationEntry, _a1 error) *ReplicationLog_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ReplicationLog_Get_Call) RunAndReturn(run func(uint64) (*replication.ReplicationEntry, error)) *ReplicationLog_Get_Call {
	_c.Call.Return(run)
	return _c
}

// GetFrom provides a mock function with given fields: startIndex, maxCount
func (_m *ReplicationLog) GetFrom(startIndex uint64, maxCount int) ([]*replication.ReplicationEntry, error) {
	ret := _m.Called(startIndex, maxCount)

	if len(ret) == 0 {
		panic("no return value specified for GetFrom")
	}

	var r0 []*replication.ReplicationEntry
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64, int) ([]*replication.ReplicationEntry, error)); ok {
		return rf(startIndex, maxCount)
	}
	if rf, ok := ret.Get(0).(func(uint64, int) []*replication.ReplicationEntry); ok {
		r0 = rf(startIndex, maxCount)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*replication.ReplicationEntry)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64, int) error); ok {
		r1 = rf(startIndex, maxCount)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ReplicationLog_GetFrom_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetFrom'
type ReplicationLog_GetFrom_Call struct {
	*mock.Call
}

// GetFrom is a helper method to define mock.On call
//   - startIndex uint64
//   - maxCount int
func (_e *ReplicationLog_Expecter) GetFrom(startIndex interface{}, maxCount interface{}) *ReplicationLog_GetFrom_Call {
	return &ReplicationLog_GetFrom_Call{Call: _e.mock.On("GetFrom", startIndex, maxCount)}
}

func (_c *ReplicationLog_GetFrom_Call) Run(run func(startIndex uint64, maxCount int)) *ReplicationLog_GetFrom_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64), args[1].(int))
	})
	return _c
}

func (_c *ReplicationLog_GetFrom_Call) Return(_a0 []*replication.ReplicationEntry, _a1 error) *ReplicationLog_GetFrom_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ReplicationLog_GetFrom_Call) RunAndReturn(run func(uint64, int) ([]*replication.ReplicationEntry, error)) *ReplicationLog_GetFrom_Call {
	_c.Call.Return(run)
	return _c
}

// GetLastIndex provides a mock function with no fields
func (_m *ReplicationLog) GetLastIndex() uint64 {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetLastIndex")
	}

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// ReplicationLog_GetLastIndex_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetLastIndex'
type ReplicationLog_GetLastIndex_Call struct {
	*mock.Call
}

// GetLastIndex is a helper method to define mock.On call
func (_e *ReplicationLog_Expecter) GetLastIndex() *ReplicationLog_GetLastIndex_Call {
	return &ReplicationLog_GetLastIndex_Call{Call: _e.mock.On("GetLastIndex")}
}

func (_c *ReplicationLog_GetLastIndex_Call) Run(run func()) *ReplicationLog_GetLastIndex_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ReplicationLog_GetLastIndex_Call) Return(_a0 uint64) *ReplicationLog_GetLastIndex_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ReplicationLog_GetLastIndex_Call) RunAndReturn(run func() uint64) *ReplicationLog_GetLastIndex_Call {
	_c.Call.Return(run)
	return _c
}

// GetLastTerm provides a mock function with no fields
func (_m *ReplicationLog) GetLastTerm() uint64 {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetLastTerm")
	}

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// ReplicationLog_GetLastTerm_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetLastTerm'
type ReplicationLog_GetLastTerm_Call struct {
	*mock.Call
}

// GetLastTerm is a helper method to define mock.On call
func (_e *ReplicationLog_Expecter) GetLastTerm() *ReplicationLog_GetLastTerm_Call {
	return &ReplicationLog_GetLastTerm_Call{Call: _e.mock.On("GetLastTerm")}
}

func (_c *ReplicationLog_GetLastTerm_Call) Run(run func()) *ReplicationLog_GetLastTerm_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ReplicationLog_GetLastTerm_Call) Return(_a0 uint64) *ReplicationLog_GetLastTerm_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ReplicationLog_GetLastTerm_Call) RunAndReturn(run func() uint64) *ReplicationLog_GetLastTerm_Call {
	_c.Call.Return(run)
	return _c
}

// Truncate provides a mock function with given fields: index
func (_m *ReplicationLog) Truncate(index uint64) error {
	ret := _m.Called(index)

	if len(ret) == 0 {
		panic("no return value specified for Truncate")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(uint64) error); ok {
		r0 = rf(index)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ReplicationLog_Truncate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Truncate'
type ReplicationLog_Truncate_Call struct {
	*mock.Call
}

// Truncate is a helper method to define mock.On call
//   - index uint64
func (_e *ReplicationLog_Expecter) Truncate(index interface{}) *ReplicationLog_Truncate_Call {
	return &ReplicationLog_Truncate_Call{Call: _e.mock.On("Truncate", index)}
}

func (_c *ReplicationLog_Truncate_Call) Run(run func(index uint64)) *ReplicationLog_Truncate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *ReplicationLog_Truncate_Call) Return(_a0 error) *ReplicationLog_Truncate_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ReplicationLog_Truncate_Call) RunAndReturn(run func(uint64) error) *ReplicationLog_Truncate_Call {
	_c.Call.Return(run)
	return _c
}

// NewReplicationLog creates a new instance of ReplicationLog. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewReplicationLog(t interface {
	mock.TestingT
	Cleanup(func())
}) *ReplicationLog {
	mock := &ReplicationLog{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
