// Code generated by mockery v1.0.0. DO NOT EDIT.
package mocks

import mock "github.com/stretchr/testify/mock"
import token "github.com/hyperledger/fabric/token"

// SigningIdentity is an autogenerated mock type for the SigningIdentity type
type SigningIdentity struct {
	mock.Mock
}

// GetPublicVersion provides a mock function with given fields:
func (_m *SigningIdentity) GetPublicVersion() token.Identity {
	ret := _m.Called()

	var r0 token.Identity
	if rf, ok := ret.Get(0).(func() token.Identity); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(token.Identity)
		}
	}

	return r0
}

// Serialize provides a mock function with given fields:
func (_m *SigningIdentity) Serialize() ([]byte, error) {
	ret := _m.Called()

	var r0 []byte
	if rf, ok := ret.Get(0).(func() []byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Sign provides a mock function with given fields: msg
func (_m *SigningIdentity) Sign(msg []byte) ([]byte, error) {
	ret := _m.Called(msg)

	var r0 []byte
	if rf, ok := ret.Get(0).(func([]byte) []byte); ok {
		r0 = rf(msg)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(msg)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}