package v1

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bitbucket.org/atlassian/logevent"
	"bitbucket.org/atlassian/vpcflow-diffd/pkg/domain"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func newHandlerFunc(storage domain.Storage, queuer domain.Queuer, method string) http.HandlerFunc {
	handler := &DiffHandler{
		LogProvider: logevent.FromContext,
		Storage:     storage,
		Queuer:      queuer,
	}
	if method == http.MethodGet {
		return handler.Get
	}
	return handler.Post
}

func newValidRequest(method string) *http.Request {
	pStart := time.Now().Format(time.RFC3339Nano)
	pStop := time.Now().Format(time.RFC3339Nano)
	nStart := time.Now().Format(time.RFC3339Nano)
	nStop := time.Now().Format(time.RFC3339Nano)

	var r *http.Request
	if method == http.MethodGet {
		r, _ = http.NewRequest(http.MethodGet, "/", nil)
	} else {
		r, _ = http.NewRequest(http.MethodPost, "/", nil)
	}

	q := r.URL.Query()
	q.Set("previous_start", pStart)
	q.Set("previous_stop", pStop)
	q.Set("next_start", nStart)
	q.Set("next_stop", nStop)
	r.URL.RawQuery = q.Encode()
	r = r.WithContext(logevent.NewContext(context.Background(), logevent.New(logevent.Config{Output: ioutil.Discard})))

	return r
}

func TestHTTPBadRequest(t *testing.T) {
	tc := []struct {
		Name          string
		PreviousStart string
		PreviousStop  string
		NextStart     string
		NextStop      string
		Method        string
	}{
		{
			Name:          "POST_bad_previous_start",
			PreviousStart: "invalid ts",
			PreviousStop:  time.Now().Format(time.RFC3339Nano),
			NextStart:     time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			NextStop:      time.Now().Format(time.RFC3339Nano),
			Method:        http.MethodPost,
		},
		{
			Name:          "POST_bad_previous_stop",
			PreviousStart: time.Now().Format(time.RFC3339Nano),
			PreviousStop:  "invalid ts",
			NextStart:     time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			NextStop:      time.Now().Format(time.RFC3339Nano),
			Method:        http.MethodPost,
		},
		{
			Name:          "POST_bad_previous_range",
			PreviousStart: time.Now().Format(time.RFC3339Nano),
			PreviousStop:  time.Now().Add(-1 * time.Minute).Format(time.RFC3339Nano),
			NextStart:     time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			NextStop:      time.Now().Format(time.RFC3339Nano),
			Method:        http.MethodPost,
		},
		{
			Name:          "POST_bad_next_start",
			PreviousStart: time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			PreviousStop:  time.Now().Format(time.RFC3339Nano),
			NextStart:     "invalid ts",
			NextStop:      time.Now().Format(time.RFC3339Nano),
			Method:        http.MethodPost,
		},
		{
			Name:          "POST_bad_next_stop",
			PreviousStart: time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			PreviousStop:  time.Now().Format(time.RFC3339Nano),
			NextStart:     time.Now().Format(time.RFC3339Nano),
			NextStop:      "invalid ts",
			Method:        http.MethodPost,
		},
		{
			Name:          "POST_bad_next_range",
			PreviousStart: time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			PreviousStop:  time.Now().Format(time.RFC3339Nano),
			NextStart:     time.Now().Format(time.RFC3339Nano),
			NextStop:      time.Now().Add(-1 * time.Minute).Format(time.RFC3339Nano),
			Method:        http.MethodPost,
		},
		{
			Name:          "POST_next_range_before_previous",
			PreviousStart: time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			PreviousStop:  time.Now().Format(time.RFC3339Nano),
			NextStart:     time.Now().Add(-2 * time.Hour).Format(time.RFC3339Nano),
			NextStop:      time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			Method:        http.MethodPost,
		},
		{
			Name:          "GET_bad_previous_start",
			PreviousStart: "invalid ts",
			PreviousStop:  time.Now().Format(time.RFC3339Nano),
			NextStart:     time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			NextStop:      time.Now().Format(time.RFC3339Nano),
			Method:        http.MethodGet,
		},
		{
			Name:          "GET_bad_previous_stop",
			PreviousStart: time.Now().Format(time.RFC3339Nano),
			PreviousStop:  "invalid ts",
			NextStart:     time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			NextStop:      time.Now().Format(time.RFC3339Nano),
			Method:        http.MethodGet,
		},
		{
			Name:          "GET_bad_previous_range",
			PreviousStart: time.Now().Format(time.RFC3339Nano),
			PreviousStop:  time.Now().Add(-1 * time.Minute).Format(time.RFC3339Nano),
			NextStart:     time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			NextStop:      time.Now().Format(time.RFC3339Nano),
			Method:        http.MethodGet,
		},
		{
			Name:          "GET_bad_next_start",
			PreviousStart: time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			PreviousStop:  time.Now().Format(time.RFC3339Nano),
			NextStart:     "invalid ts",
			NextStop:      time.Now().Format(time.RFC3339Nano),
			Method:        http.MethodGet,
		},
		{
			Name:          "GET_bad_next_stop",
			PreviousStart: time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			PreviousStop:  time.Now().Format(time.RFC3339Nano),
			NextStart:     time.Now().Format(time.RFC3339Nano),
			NextStop:      "invalid ts",
			Method:        http.MethodGet,
		},
		{
			Name:          "GET_bad_next_range",
			PreviousStart: time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			PreviousStop:  time.Now().Format(time.RFC3339Nano),
			NextStart:     time.Now().Format(time.RFC3339Nano),
			NextStop:      time.Now().Add(-1 * time.Minute).Format(time.RFC3339Nano),
			Method:        http.MethodGet,
		},
		{
			Name:          "GET_next_range_before_previous",
			PreviousStart: time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			PreviousStop:  time.Now().Format(time.RFC3339Nano),
			NextStart:     time.Now().Add(-2 * time.Hour).Format(time.RFC3339Nano),
			NextStop:      time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano),
			Method:        http.MethodGet,
		},
	}

	for _, tt := range tc {
		t.Run(tt.Name, func(t *testing.T) {
			r, _ := http.NewRequest(tt.Method, "/", nil)
			w := httptest.NewRecorder()

			q := r.URL.Query()
			q.Set("previous_start", tt.PreviousStart)
			q.Set("previous_stop", tt.PreviousStop)
			q.Set("next_start", tt.NextStart)
			q.Set("next_stop", tt.NextStop)
			r.URL.RawQuery = q.Encode()
			r = r.WithContext(logevent.NewContext(context.Background(), logevent.New(logevent.Config{Output: ioutil.Discard})))

			newHandlerFunc(nil, nil, tt.Method)(w, r)

			assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
		})
	}
}

func TestGetStorageErrors(t *testing.T) {
	tc := []struct {
		Name               string
		Error              error
		ExpectedStatusCode int
	}{
		{
			Name:               "GET_in_progress",
			Error:              domain.ErrInProgress{},
			ExpectedStatusCode: http.StatusNoContent,
		},
		{
			Name:               "GET_not_found",
			Error:              domain.ErrNotFound{},
			ExpectedStatusCode: http.StatusNotFound,
		},
		{
			Name:               "GET_unknown",
			Error:              errors.New("oops"),
			ExpectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tc {
		t.Run(tt.Name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			r := newValidRequest(http.MethodGet)
			w := httptest.NewRecorder()

			storageMock := NewMockStorage(ctrl)
			storageMock.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, tt.Error)

			h := DiffHandler{
				LogProvider: logevent.FromContext,
				Storage:     storageMock,
			}
			h.Get(w, r)

			assert.Equal(t, tt.ExpectedStatusCode, w.Result().StatusCode)
		})
	}
}

func TestGetHappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := newValidRequest(http.MethodGet)
	w := httptest.NewRecorder()

	data := "this is the diff you're looking for"
	readCloser := ioutil.NopCloser(bytes.NewReader([]byte(data)))

	storageMock := NewMockStorage(ctrl)
	storageMock.EXPECT().Get(gomock.Any(), gomock.Any()).Return(readCloser, nil)

	h := DiffHandler{
		LogProvider: logevent.FromContext,
		Storage:     storageMock,
	}
	h.Get(w, r)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	body := w.Result().Body
	defer body.Close()

	result, _ := ioutil.ReadAll(body)
	assert.Equal(t, data, string(result))
}

func TestPostConflictInProgress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := newValidRequest(http.MethodPost)
	w := httptest.NewRecorder()

	storageMock := NewMockStorage(ctrl)
	storageMock.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, domain.ErrInProgress{})

	h := DiffHandler{
		LogProvider: logevent.FromContext,
		Storage:     storageMock,
	}
	h.Post(w, r)

	assert.Equal(t, http.StatusConflict, w.Result().StatusCode)
}

func TestPostConflictDiffCreated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := newValidRequest(http.MethodPost)
	w := httptest.NewRecorder()

	storageMock := NewMockStorage(ctrl)
	storageMock.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(true, nil)

	h := DiffHandler{
		LogProvider: logevent.FromContext,
		Storage:     storageMock,
	}
	h.Post(w, r)

	assert.Equal(t, http.StatusConflict, w.Result().StatusCode)
}

func TestPostStorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := newValidRequest(http.MethodPost)
	w := httptest.NewRecorder()

	storageMock := NewMockStorage(ctrl)
	storageMock.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, errors.New("oops"))

	h := DiffHandler{
		LogProvider: logevent.FromContext,
		Storage:     storageMock,
	}
	h.Post(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestPostQueueError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := newValidRequest(http.MethodPost)
	w := httptest.NewRecorder()

	storageMock := NewMockStorage(ctrl)
	storageMock.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil)
	queuerMock := NewMockQueuer(ctrl)
	queuerMock.EXPECT().Queue(gomock.Any(), gomock.Any()).Return(errors.New("oops"))

	h := DiffHandler{
		LogProvider: logevent.FromContext,
		Storage:     storageMock,
		Queuer:      queuerMock,
	}
	h.Post(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestPostHappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := newValidRequest(http.MethodPost)
	w := httptest.NewRecorder()

	storageMock := NewMockStorage(ctrl)
	storageMock.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil)
	queuerMock := NewMockQueuer(ctrl)
	queuerMock.EXPECT().Queue(gomock.Any(), gomock.Any()).Return(nil)
	markerMock := NewMockMarker(ctrl)
	markerMock.EXPECT().Mark(gomock.Any(), gomock.Any()).Return(nil)

	h := DiffHandler{
		LogProvider: logevent.FromContext,
		Storage:     storageMock,
		Queuer:      queuerMock,
		Marker:      markerMock,
	}
	h.Post(w, r)

	assert.Equal(t, http.StatusAccepted, w.Result().StatusCode)
}

func TestPostUnsuccessfulMark(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := newValidRequest(http.MethodPost)
	w := httptest.NewRecorder()

	storageMock := NewMockStorage(ctrl)
	storageMock.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil)
	queuerMock := NewMockQueuer(ctrl)
	queuerMock.EXPECT().Queue(gomock.Any(), gomock.Any()).Return(nil)
	markerMock := NewMockMarker(ctrl)
	markerMock.EXPECT().Mark(gomock.Any(), gomock.Any()).Return(errors.New("OOPS"))

	h := DiffHandler{
		LogProvider: logevent.FromContext,
		Storage:     storageMock,
		Queuer:      queuerMock,
		Marker:      markerMock,
	}
	h.Post(w, r)

	// Shouldn't blow up
	assert.Equal(t, http.StatusAccepted, w.Result().StatusCode)
}
