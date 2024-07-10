// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	metrics "github.com/smartcontractkit/chainlink-solana/pkg/monitoring/metrics"
	mock "github.com/stretchr/testify/mock"
)

// NetworkFees is an autogenerated mock type for the NetworkFees type
type NetworkFees struct {
	mock.Mock
}

// Cleanup provides a mock function with given fields:
func (_m *NetworkFees) Cleanup() {
	_m.Called()
}

// Set provides a mock function with given fields: slot, chain
func (_m *NetworkFees) Set(slot metrics.NetworkFeesInput, chain string) {
	_m.Called(slot, chain)
}

type mockConstructorTestingTNewNetworkFees interface {
	mock.TestingT
	Cleanup(func())
}

// NewNetworkFees creates a new instance of NetworkFees. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewNetworkFees(t mockConstructorTestingTNewNetworkFees) *NetworkFees {
	mock := &NetworkFees{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}