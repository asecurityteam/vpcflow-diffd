// Automatically generated by MockGen. DO NOT EDIT!
// Source: ./pkg/domain/grapher.go

package differ

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	io "io"
	time "time"
)

// Mock of Grapher interface
type MockGrapher struct {
	ctrl     *gomock.Controller
	recorder *_MockGrapherRecorder
}

// Recorder for MockGrapher (not exported)
type _MockGrapherRecorder struct {
	mock *MockGrapher
}

func NewMockGrapher(ctrl *gomock.Controller) *MockGrapher {
	mock := &MockGrapher{ctrl: ctrl}
	mock.recorder = &_MockGrapherRecorder{mock}
	return mock
}

func (_m *MockGrapher) EXPECT() *_MockGrapherRecorder {
	return _m.recorder
}

func (_m *MockGrapher) Graph(ctx context.Context, start time.Time, stop time.Time) (io.ReadCloser, error) {
	ret := _m.ctrl.Call(_m, "Graph", ctx, start, stop)
	ret0, _ := ret[0].(io.ReadCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockGrapherRecorder) Graph(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Graph", arg0, arg1, arg2)
}
