// Code generated by mockery v1.1.0. DO NOT EDIT.

package aws

import (
	context "context"

	ec2 "github.com/aws/aws-sdk-go/service/ec2"
	mock "github.com/stretchr/testify/mock"

	request "github.com/aws/aws-sdk-go/aws/request"
)

// mockEc2Client is an autogenerated mock type for the ec2Client type
type mockEc2Client struct {
	mock.Mock
}

// DescribeNetworkInterfacesWithContext provides a mock function with given fields: _a0, _a1, _a2
func (_m *mockEc2Client) DescribeNetworkInterfacesWithContext(_a0 context.Context, _a1 *ec2.DescribeNetworkInterfacesInput, _a2 ...request.Option) (*ec2.DescribeNetworkInterfacesOutput, error) {
	_va := make([]interface{}, len(_a2))
	for _i := range _a2 {
		_va[_i] = _a2[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _a0, _a1)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *ec2.DescribeNetworkInterfacesOutput
	if rf, ok := ret.Get(0).(func(context.Context, *ec2.DescribeNetworkInterfacesInput, ...request.Option) *ec2.DescribeNetworkInterfacesOutput); ok {
		r0 = rf(_a0, _a1, _a2...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ec2.DescribeNetworkInterfacesOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *ec2.DescribeNetworkInterfacesInput, ...request.Option) error); ok {
		r1 = rf(_a0, _a1, _a2...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
