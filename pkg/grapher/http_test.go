package grapher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestPOSTRequstError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRT := NewMockRoundTripper(ctrl)
	setClientExpectations(mockRT, http.MethodPost, errors.New(""))
	_, err := execute(context.Background(), mockRT)
	assert.NotNil(t, err)
}

func TestPOST5XX(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRT := NewMockRoundTripper(ctrl)
	setClientExpectations(mockRT, http.MethodPost, nil, response{statusCode: 500})
	_, err := execute(context.Background(), mockRT)
	assert.NotNil(t, err)
}

func TestGraphExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRT := NewMockRoundTripper(ctrl)
	body := "this is a graph"
	setClientExpectations(mockRT, http.MethodPost, nil, response{statusCode: 409})
	setClientExpectations(mockRT, http.MethodGet, nil, response{body: body, statusCode: 200})
	output, err := execute(context.Background(), mockRT)
	assert.Nil(t, err)
	defer output.Close()
	data, _ := ioutil.ReadAll(output)
	assert.Equal(t, body, string(data))
}

func TestGraphGETRequestError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRT := NewMockRoundTripper(ctrl)
	setClientExpectations(mockRT, http.MethodPost, nil, response{statusCode: 202})
	setClientExpectations(mockRT, http.MethodGet, errors.New(""))
	_, err := execute(context.Background(), mockRT)
	assert.NotNil(t, err)
}

func TestGET5XX(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRT := NewMockRoundTripper(ctrl)
	setClientExpectations(mockRT, http.MethodPost, nil, response{statusCode: 202})
	setClientExpectations(mockRT, http.MethodGet, nil, response{statusCode: 500})
	_, err := execute(context.Background(), mockRT)
	assert.NotNil(t, err)
}

func TestGraphContextCancelled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx, cancel := context.WithCancel(context.Background())
	mockRT := NewMockRoundTripper(ctrl)
	setClientExpectations(mockRT, http.MethodPost, nil, response{statusCode: 409})
	setClientExpectations(mockRT, http.MethodGet, nil, response{statusCode: 204})
	mockRT.EXPECT().RoundTrip(&requestMethodMatcher{method: http.MethodGet}).Do(func(r *http.Request) {
		cancel()
	}).Return(&http.Response{StatusCode: 204, Body: ioutil.NopCloser(bytes.NewReader([]byte("")))}, nil)
	_, err := execute(ctx, mockRT)
	assert.NotNil(t, err)
}

func TestGraphRetriesSuceeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	pollAttempts := 3
	body := "this is another graph"
	mockRT := NewMockRoundTripper(ctrl)
	resps := make([]response, pollAttempts)
	for i := 0; i < pollAttempts-1; i++ {
		resps[i] = response{statusCode: 204}
	}
	resps[pollAttempts-1] = response{statusCode: 200, body: body}
	setClientExpectations(mockRT, http.MethodPost, nil, response{statusCode: 409})
	setClientExpectations(mockRT, http.MethodGet, nil, resps...)
	output, err := execute(context.Background(), mockRT)
	assert.Nil(t, err)
	defer output.Close()
	data, _ := ioutil.ReadAll(output)
	assert.Equal(t, body, string(data))
}

func execute(ctx context.Context, rt http.RoundTripper) (io.ReadCloser, error) {
	u, _ := url.Parse("http://host")
	stop := time.Now()
	start := stop.Add(-1 * time.Minute)
	c := HTTP{
		Endpoint:        u,
		Client:          &http.Client{Transport: rt},
		PollTimeout:     time.Minute,
		PollingInterval: time.Duration(-1),
	}
	return c.Graph(ctx, start, stop)
}

type response struct {
	statusCode int
	body       string
}

type requestMethodMatcher struct {
	method string
}

func (rm *requestMethodMatcher) Matches(x interface{}) bool {
	req, ok := x.(*http.Request)
	if !ok {
		return false
	}
	return req.Method == rm.method
}

func (rm *requestMethodMatcher) String() string {
	return fmt.Sprintf("Request method matches %s", rm.method)
}

func setClientExpectations(mock *MockRoundTripper, method string, err error, resps ...response) {
	if err != nil {
		mock.EXPECT().RoundTrip(&requestMethodMatcher{method: method}).Return(nil, err)
		return
	}
	for _, resp := range resps {
		payload := ioutil.NopCloser(bytes.NewReader([]byte(resp.body)))
		mock.EXPECT().RoundTrip(&requestMethodMatcher{method: method}).Return(&http.Response{
			Body:       payload,
			StatusCode: resp.statusCode,
		}, nil)
	}
}
