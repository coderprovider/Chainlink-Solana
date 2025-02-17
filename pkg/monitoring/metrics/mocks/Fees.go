// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	metrics "github.com/smartcontractkit/chainlink-solana/pkg/monitoring/metrics"
	fees "github.com/smartcontractkit/chainlink-solana/pkg/solana/fees"

	mock "github.com/stretchr/testify/mock"
)

// Fees is an autogenerated mock type for the Fees type
type Fees struct {
	mock.Mock
}

// Cleanup provides a mock function with given fields: feedInput
func (_m *Fees) Cleanup(feedInput metrics.FeedInput) {
	_m.Called(feedInput)
}

// Set provides a mock function with given fields: txFee, computeUnitPrice, feedInput
func (_m *Fees) Set(txFee uint64, computeUnitPrice fees.ComputeUnitPrice, feedInput metrics.FeedInput) {
	_m.Called(txFee, computeUnitPrice, feedInput)
}

type mockConstructorTestingTNewFees interface {
	mock.TestingT
	Cleanup(func())
}

// NewFees creates a new instance of Fees. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewFees(t mockConstructorTestingTNewFees) *Fees {
	mock := &Fees{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
