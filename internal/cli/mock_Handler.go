// Code generated by mockery v1.1.0. DO NOT EDIT.

package cli

import mock "github.com/stretchr/testify/mock"

// MockHandler is an autogenerated mock type for the Handler type
type MockHandler struct {
	mock.Mock
}

// Execute provides a mock function with given fields: context
func (_m *MockHandler) Execute(context *Context) error {
	ret := _m.Called(context)

	var r0 error
	if rf, ok := ret.Get(0).(func(*Context) error); ok {
		r0 = rf(context)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
