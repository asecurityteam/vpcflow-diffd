// Automatically generated by MockGen. DO NOT EDIT!
// Source: ./pkg/domain/queuer.go

package v1

import (
	context "context"

	"github.com/asecurityteam/vpcflow-diffd/pkg/domain"
	gomock "github.com/golang/mock/gomock"
)

// Mock of Queuer interface
type MockQueuer struct {
	ctrl     *gomock.Controller
	recorder *_MockQueuerRecorder
}

// Recorder for MockQueuer (not exported)
type _MockQueuerRecorder struct {
	mock *MockQueuer
}

func NewMockQueuer(ctrl *gomock.Controller) *MockQueuer {
	mock := &MockQueuer{ctrl: ctrl}
	mock.recorder = &_MockQueuerRecorder{mock}
	return mock
}

func (_m *MockQueuer) EXPECT() *_MockQueuerRecorder {
	return _m.recorder
}

func (_m *MockQueuer) Queue(ctx context.Context, d domain.Diff) error {
	ret := _m.ctrl.Call(_m, "Queue", ctx, d)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockQueuerRecorder) Queue(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Queue", arg0, arg1)
}
