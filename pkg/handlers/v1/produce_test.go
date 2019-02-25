package v1

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/asecurityteam/logevent"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	payloadTpl = `{"id":"%s","previousStart":"%s","previousStop":"%s","nextStart":"%s","nextStop":"%s"}`
	diffID     = "diffid"
)

func newProduceRequest() *http.Request {
	pStart := time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano)
	pStop := time.Now().Format(time.RFC3339Nano)
	nStart := time.Now().Add(time.Hour).Format(time.RFC3339Nano)
	nStop := time.Now().Add(2 * time.Hour).Format(time.RFC3339Nano)
	payload := fmt.Sprintf(payloadTpl, diffID, pStart, pStop, nStart, nStop)
	r, _ := http.NewRequest(http.MethodPost, "/", ioutil.NopCloser(bytes.NewReader([]byte(payload))))
	return r.WithContext(logevent.NewContext(context.Background(), logevent.New(logevent.Config{Output: ioutil.Discard})))
}

func TestProduceBadRequst(t *testing.T) {
	tc := []struct {
		Name          string
		ID            string
		PreviousStart string
		PreviousStop  string
		NextStart     string
		NextStop      string
	}{
		{
			Name: "invalid_json",
			ID:   "\"",
		},
		{
			Name:          "invalid_id",
			ID:            "",
			PreviousStart: time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			PreviousStop:  time.Now().Format(time.RFC3339Nano),
			NextStart:     time.Now().Add(time.Hour).Format(time.RFC3339Nano),
			NextStop:      time.Now().Add(2 * time.Hour).Format(time.RFC3339Nano),
		},
		{
			Name:          "invalid_previous_start",
			ID:            diffID,
			PreviousStart: "",
			PreviousStop:  time.Now().Format(time.RFC3339Nano),
			NextStart:     time.Now().Add(time.Hour).Format(time.RFC3339Nano),
			NextStop:      time.Now().Add(2 * time.Hour).Format(time.RFC3339Nano),
		},
		{
			Name:          "invalid_previous_stop",
			ID:            diffID,
			PreviousStart: time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			PreviousStop:  "",
			NextStart:     time.Now().Add(time.Hour).Format(time.RFC3339Nano),
			NextStop:      time.Now().Add(2 * time.Hour).Format(time.RFC3339Nano),
		},
		{
			Name:          "invalid_previous_range",
			ID:            diffID,
			PreviousStart: time.Now().Format(time.RFC3339Nano),
			PreviousStop:  time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			NextStart:     time.Now().Add(time.Hour).Format(time.RFC3339Nano),
			NextStop:      time.Now().Add(2 * time.Hour).Format(time.RFC3339Nano),
		},
		{
			Name:          "invalid_next_start",
			ID:            diffID,
			PreviousStart: time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			PreviousStop:  time.Now().Format(time.RFC3339Nano),
			NextStart:     "",
			NextStop:      time.Now().Add(2 * time.Hour).Format(time.RFC3339Nano),
		},
		{
			Name:          "invalid_next_stop",
			ID:            diffID,
			PreviousStart: time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			PreviousStop:  time.Now().Format(time.RFC3339Nano),
			NextStart:     time.Now().Add(time.Hour).Format(time.RFC3339Nano),
			NextStop:      "",
		},
		{
			Name:          "invalid_next_range",
			ID:            diffID,
			PreviousStart: time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			PreviousStop:  time.Now().Format(time.RFC3339Nano),
			NextStart:     time.Now().Add(3 * time.Hour).Format(time.RFC3339Nano),
			NextStop:      time.Now().Add(2 * time.Hour).Format(time.RFC3339Nano),
		},
		{
			Name:          "invalid_diff_range",
			ID:            diffID,
			NextStart:     time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			NextStop:      time.Now().Format(time.RFC3339Nano),
			PreviousStart: time.Now().Add(1 * time.Hour).Format(time.RFC3339Nano),
			PreviousStop:  time.Now().Add(2 * time.Hour).Format(time.RFC3339Nano),
		},
	}

	for _, tt := range tc {
		t.Run(tt.Name, func(t *testing.T) {
			payload := fmt.Sprintf(payloadTpl, tt.ID, tt.PreviousStart, tt.PreviousStop, tt.NextStart, tt.NextStop)
			r, _ := http.NewRequest(http.MethodPost, "/", ioutil.NopCloser(bytes.NewReader([]byte(payload))))
			r = r.WithContext(logevent.NewContext(context.Background(), logevent.New(logevent.Config{Output: ioutil.Discard})))
			w := httptest.NewRecorder()
			handler := &Produce{
				LogProvider: logevent.FromContext,
			}
			handler.ServeHTTP(w, r)
			assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
		})
	}
}

func TestProduceDiffFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDiffer := NewMockDiffer(ctrl)
	mockDiffer.EXPECT().Diff(gomock.Any(), gomock.Any()).Return(nil, errors.New(""))

	r := newProduceRequest()
	w := httptest.NewRecorder()
	handler := &Produce{
		LogProvider: logevent.FromContext,
		Differ:      mockDiffer,
	}
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestProduceStoreFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDiffer := NewMockDiffer(ctrl)
	mockDiffer.EXPECT().Diff(gomock.Any(), gomock.Any()).Return(ioutil.NopCloser(bytes.NewReader([]byte(""))), nil)
	mockStorage := NewMockStorage(ctrl)
	mockStorage.EXPECT().Store(gomock.Any(), diffID, gomock.Any()).Return(errors.New(""))

	r := newProduceRequest()
	w := httptest.NewRecorder()
	handler := &Produce{
		LogProvider: logevent.FromContext,
		Differ:      mockDiffer,
		Storage:     mockStorage,
	}
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestProduceUnmarkFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDiffer := NewMockDiffer(ctrl)
	mockDiffer.EXPECT().Diff(gomock.Any(), gomock.Any()).Return(ioutil.NopCloser(bytes.NewReader([]byte(""))), nil)
	mockStorage := NewMockStorage(ctrl)
	mockStorage.EXPECT().Store(gomock.Any(), diffID, gomock.Any()).Return(nil)
	mockMarker := NewMockMarker(ctrl)
	mockMarker.EXPECT().Unmark(gomock.Any(), diffID).Return(errors.New(""))

	r := newProduceRequest()
	w := httptest.NewRecorder()
	handler := &Produce{
		LogProvider: logevent.FromContext,
		Differ:      mockDiffer,
		Storage:     mockStorage,
		Marker:      mockMarker,
	}
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestProduce(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDiffer := NewMockDiffer(ctrl)
	mockDiffer.EXPECT().Diff(gomock.Any(), gomock.Any()).Return(ioutil.NopCloser(bytes.NewReader([]byte(""))), nil)
	mockStorage := NewMockStorage(ctrl)
	mockStorage.EXPECT().Store(gomock.Any(), diffID, gomock.Any()).Return(nil)
	mockMarker := NewMockMarker(ctrl)
	mockMarker.EXPECT().Unmark(gomock.Any(), diffID).Return(nil)

	r := newProduceRequest()
	w := httptest.NewRecorder()
	handler := &Produce{
		LogProvider: logevent.FromContext,
		Differ:      mockDiffer,
		Storage:     mockStorage,
		Marker:      mockMarker,
	}
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusNoContent, w.Result().StatusCode)
}
