// Code generated by mockery v1.1.0. DO NOT EDIT.

package env

import mock "github.com/stretchr/testify/mock"

// MockEnv is an autogenerated mock type for the Env type
type MockEnv struct {
	mock.Mock
}

// Get provides a mock function with given fields: key
func (_m *MockEnv) Get(key string) string {
	ret := _m.Called(key)

	var r0 string
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
