// Code generated by mockery v2.50.0. DO NOT EDIT.

package mocks

import (
	cache "github.com/fyerfyer/fyer-cache/cache"
	mock "github.com/stretchr/testify/mock"
)

// ClusterNode is an autogenerated mock type for the ClusterNode type
type ClusterNode struct {
	mock.Mock
}

type ClusterNode_Expecter struct {
	mock *mock.Mock
}

func (_m *ClusterNode) EXPECT() *ClusterNode_Expecter {
	return &ClusterNode_Expecter{mock: &_m.Mock}
}

// Events provides a mock function with no fields
func (_m *ClusterNode) Events() <-chan cache.ClusterEvent {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Events")
	}

	var r0 <-chan cache.ClusterEvent
	if rf, ok := ret.Get(0).(func() <-chan cache.ClusterEvent); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan cache.ClusterEvent)
		}
	}

	return r0
}

// ClusterNode_Events_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Events'
type ClusterNode_Events_Call struct {
	*mock.Call
}

// Events is a helper method to define mock.On call
func (_e *ClusterNode_Expecter) Events() *ClusterNode_Events_Call {
	return &ClusterNode_Events_Call{Call: _e.mock.On("Events")}
}

func (_c *ClusterNode_Events_Call) Run(run func()) *ClusterNode_Events_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ClusterNode_Events_Call) Return(_a0 <-chan cache.ClusterEvent) *ClusterNode_Events_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ClusterNode_Events_Call) RunAndReturn(run func() <-chan cache.ClusterEvent) *ClusterNode_Events_Call {
	_c.Call.Return(run)
	return _c
}

// IsCoordinator provides a mock function with no fields
func (_m *ClusterNode) IsCoordinator() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for IsCoordinator")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// ClusterNode_IsCoordinator_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsCoordinator'
type ClusterNode_IsCoordinator_Call struct {
	*mock.Call
}

// IsCoordinator is a helper method to define mock.On call
func (_e *ClusterNode_Expecter) IsCoordinator() *ClusterNode_IsCoordinator_Call {
	return &ClusterNode_IsCoordinator_Call{Call: _e.mock.On("IsCoordinator")}
}

func (_c *ClusterNode_IsCoordinator_Call) Run(run func()) *ClusterNode_IsCoordinator_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ClusterNode_IsCoordinator_Call) Return(_a0 bool) *ClusterNode_IsCoordinator_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ClusterNode_IsCoordinator_Call) RunAndReturn(run func() bool) *ClusterNode_IsCoordinator_Call {
	_c.Call.Return(run)
	return _c
}

// Join provides a mock function with given fields: seedNodeAddr
func (_m *ClusterNode) Join(seedNodeAddr string) error {
	ret := _m.Called(seedNodeAddr)

	if len(ret) == 0 {
		panic("no return value specified for Join")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(seedNodeAddr)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ClusterNode_Join_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Join'
type ClusterNode_Join_Call struct {
	*mock.Call
}

// Join is a helper method to define mock.On call
//   - seedNodeAddr string
func (_e *ClusterNode_Expecter) Join(seedNodeAddr interface{}) *ClusterNode_Join_Call {
	return &ClusterNode_Join_Call{Call: _e.mock.On("Join", seedNodeAddr)}
}

func (_c *ClusterNode_Join_Call) Run(run func(seedNodeAddr string)) *ClusterNode_Join_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *ClusterNode_Join_Call) Return(_a0 error) *ClusterNode_Join_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ClusterNode_Join_Call) RunAndReturn(run func(string) error) *ClusterNode_Join_Call {
	_c.Call.Return(run)
	return _c
}

// Leave provides a mock function with no fields
func (_m *ClusterNode) Leave() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Leave")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ClusterNode_Leave_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Leave'
type ClusterNode_Leave_Call struct {
	*mock.Call
}

// Leave is a helper method to define mock.On call
func (_e *ClusterNode_Expecter) Leave() *ClusterNode_Leave_Call {
	return &ClusterNode_Leave_Call{Call: _e.mock.On("Leave")}
}

func (_c *ClusterNode_Leave_Call) Run(run func()) *ClusterNode_Leave_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ClusterNode_Leave_Call) Return(_a0 error) *ClusterNode_Leave_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ClusterNode_Leave_Call) RunAndReturn(run func() error) *ClusterNode_Leave_Call {
	_c.Call.Return(run)
	return _c
}

// Members provides a mock function with no fields
func (_m *ClusterNode) Members() []cache.NodeInfo {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Members")
	}

	var r0 []cache.NodeInfo
	if rf, ok := ret.Get(0).(func() []cache.NodeInfo); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]cache.NodeInfo)
		}
	}

	return r0
}

// ClusterNode_Members_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Members'
type ClusterNode_Members_Call struct {
	*mock.Call
}

// Members is a helper method to define mock.On call
func (_e *ClusterNode_Expecter) Members() *ClusterNode_Members_Call {
	return &ClusterNode_Members_Call{Call: _e.mock.On("Members")}
}

func (_c *ClusterNode_Members_Call) Run(run func()) *ClusterNode_Members_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ClusterNode_Members_Call) Return(_a0 []cache.NodeInfo) *ClusterNode_Members_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ClusterNode_Members_Call) RunAndReturn(run func() []cache.NodeInfo) *ClusterNode_Members_Call {
	_c.Call.Return(run)
	return _c
}

// Start provides a mock function with no fields
func (_m *ClusterNode) Start() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Start")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ClusterNode_Start_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Start'
type ClusterNode_Start_Call struct {
	*mock.Call
}

// Start is a helper method to define mock.On call
func (_e *ClusterNode_Expecter) Start() *ClusterNode_Start_Call {
	return &ClusterNode_Start_Call{Call: _e.mock.On("Start")}
}

func (_c *ClusterNode_Start_Call) Run(run func()) *ClusterNode_Start_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ClusterNode_Start_Call) Return(_a0 error) *ClusterNode_Start_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ClusterNode_Start_Call) RunAndReturn(run func() error) *ClusterNode_Start_Call {
	_c.Call.Return(run)
	return _c
}

// Stop provides a mock function with no fields
func (_m *ClusterNode) Stop() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Stop")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ClusterNode_Stop_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Stop'
type ClusterNode_Stop_Call struct {
	*mock.Call
}

// Stop is a helper method to define mock.On call
func (_e *ClusterNode_Expecter) Stop() *ClusterNode_Stop_Call {
	return &ClusterNode_Stop_Call{Call: _e.mock.On("Stop")}
}

func (_c *ClusterNode_Stop_Call) Run(run func()) *ClusterNode_Stop_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ClusterNode_Stop_Call) Return(_a0 error) *ClusterNode_Stop_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ClusterNode_Stop_Call) RunAndReturn(run func() error) *ClusterNode_Stop_Call {
	_c.Call.Return(run)
	return _c
}

// NewClusterNode creates a new instance of ClusterNode. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewClusterNode(t interface {
	mock.TestingT
	Cleanup(func())
}) *ClusterNode {
	mock := &ClusterNode{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
