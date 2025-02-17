// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	context "context"

	rpc "github.com/gagliardetto/solana-go/rpc"
	mock "github.com/stretchr/testify/mock"

	solana "github.com/gagliardetto/solana-go"
)

// ReaderWriter is an autogenerated mock type for the ReaderWriter type
type ReaderWriter struct {
	mock.Mock
}

// Balance provides a mock function with given fields: addr
func (_m *ReaderWriter) Balance(addr solana.PublicKey) (uint64, error) {
	ret := _m.Called(addr)

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(solana.PublicKey) (uint64, error)); ok {
		return rf(addr)
	}
	if rf, ok := ret.Get(0).(func(solana.PublicKey) uint64); ok {
		r0 = rf(addr)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(solana.PublicKey) error); ok {
		r1 = rf(addr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChainID provides a mock function with given fields:
func (_m *ReaderWriter) ChainID() (string, error) {
	ret := _m.Called()

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func() (string, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAccountInfoWithOpts provides a mock function with given fields: ctx, addr, opts
func (_m *ReaderWriter) GetAccountInfoWithOpts(ctx context.Context, addr solana.PublicKey, opts *rpc.GetAccountInfoOpts) (*rpc.GetAccountInfoResult, error) {
	ret := _m.Called(ctx, addr, opts)

	var r0 *rpc.GetAccountInfoResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, solana.PublicKey, *rpc.GetAccountInfoOpts) (*rpc.GetAccountInfoResult, error)); ok {
		return rf(ctx, addr, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, solana.PublicKey, *rpc.GetAccountInfoOpts) *rpc.GetAccountInfoResult); ok {
		r0 = rf(ctx, addr, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rpc.GetAccountInfoResult)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, solana.PublicKey, *rpc.GetAccountInfoOpts) error); ok {
		r1 = rf(ctx, addr, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetFeeForMessage provides a mock function with given fields: msg
func (_m *ReaderWriter) GetFeeForMessage(msg string) (uint64, error) {
	ret := _m.Called(msg)

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (uint64, error)); ok {
		return rf(msg)
	}
	if rf, ok := ret.Get(0).(func(string) uint64); ok {
		r0 = rf(msg)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(msg)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLatestBlock provides a mock function with given fields:
func (_m *ReaderWriter) GetLatestBlock() (*rpc.GetBlockResult, error) {
	ret := _m.Called()

	var r0 *rpc.GetBlockResult
	var r1 error
	if rf, ok := ret.Get(0).(func() (*rpc.GetBlockResult, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *rpc.GetBlockResult); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rpc.GetBlockResult)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LatestBlockhash provides a mock function with given fields:
func (_m *ReaderWriter) LatestBlockhash() (*rpc.GetLatestBlockhashResult, error) {
	ret := _m.Called()

	var r0 *rpc.GetLatestBlockhashResult
	var r1 error
	if rf, ok := ret.Get(0).(func() (*rpc.GetLatestBlockhashResult, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *rpc.GetLatestBlockhashResult); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rpc.GetLatestBlockhashResult)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SendTx provides a mock function with given fields: ctx, tx
func (_m *ReaderWriter) SendTx(ctx context.Context, tx *solana.Transaction) (solana.Signature, error) {
	ret := _m.Called(ctx, tx)

	var r0 solana.Signature
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *solana.Transaction) (solana.Signature, error)); ok {
		return rf(ctx, tx)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *solana.Transaction) solana.Signature); ok {
		r0 = rf(ctx, tx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(solana.Signature)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *solana.Transaction) error); ok {
		r1 = rf(ctx, tx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SignatureStatuses provides a mock function with given fields: ctx, sigs
func (_m *ReaderWriter) SignatureStatuses(ctx context.Context, sigs []solana.Signature) ([]*rpc.SignatureStatusesResult, error) {
	ret := _m.Called(ctx, sigs)

	var r0 []*rpc.SignatureStatusesResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []solana.Signature) ([]*rpc.SignatureStatusesResult, error)); ok {
		return rf(ctx, sigs)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []solana.Signature) []*rpc.SignatureStatusesResult); ok {
		r0 = rf(ctx, sigs)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*rpc.SignatureStatusesResult)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []solana.Signature) error); ok {
		r1 = rf(ctx, sigs)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SimulateTx provides a mock function with given fields: ctx, tx, opts
func (_m *ReaderWriter) SimulateTx(ctx context.Context, tx *solana.Transaction, opts *rpc.SimulateTransactionOpts) (*rpc.SimulateTransactionResult, error) {
	ret := _m.Called(ctx, tx, opts)

	var r0 *rpc.SimulateTransactionResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *solana.Transaction, *rpc.SimulateTransactionOpts) (*rpc.SimulateTransactionResult, error)); ok {
		return rf(ctx, tx, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *solana.Transaction, *rpc.SimulateTransactionOpts) *rpc.SimulateTransactionResult); ok {
		r0 = rf(ctx, tx, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rpc.SimulateTransactionResult)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *solana.Transaction, *rpc.SimulateTransactionOpts) error); ok {
		r1 = rf(ctx, tx, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SlotHeight provides a mock function with given fields:
func (_m *ReaderWriter) SlotHeight() (uint64, error) {
	ret := _m.Called()

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func() (uint64, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewReaderWriter interface {
	mock.TestingT
	Cleanup(func())
}

// NewReaderWriter creates a new instance of ReaderWriter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewReaderWriter(t mockConstructorTestingTNewReaderWriter) *ReaderWriter {
	mock := &ReaderWriter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
