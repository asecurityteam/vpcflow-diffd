package grapher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const (
	queryStart = "start"
	queryStop  = "stop"
)

// HTTP is used to create a new graph
type HTTP struct {
	Client          *http.Client
	Endpoint        *url.URL
	PollTimeout     time.Duration
	PollingInterval time.Duration
}

// Graph starts a new graph job, and waits for its completion. On successful completion, Graph will return the
// graph content.
func (c *HTTP) Graph(ctx context.Context, start, stop time.Time) (io.ReadCloser, error) {
	req, err := newGraphRequest(c.Endpoint, http.MethodPost, start, stop)
	if err != nil {
		return nil, err
	}
	res, err := c.Client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	// If the status code is 202 or 409 the job is either a) scheduled b) in progress or c) created.
	// In all of these cases, we want to poll the GET endpoint for a 200. Otherwise, report an error.
	if res.StatusCode != http.StatusAccepted && res.StatusCode != http.StatusConflict {
		data, _ := ioutil.ReadAll(res.Body)
		return nil, fmt.Errorf("Received unexpected response from grapher %d: %s", res.StatusCode, data)
	}
	pollingCtx, cancel := context.WithTimeout(ctx, c.PollTimeout)
	defer cancel()
	return c.waitForGraph(pollingCtx, start, stop)
}

func (c *HTTP) waitForGraph(ctx context.Context, start, stop time.Time) (io.ReadCloser, error) {
	req, err := newGraphRequest(c.Endpoint, http.MethodGet, start, stop)
	if err != nil {
		return nil, err
	}
	var attempts int
	for {
		attempts++
		res, err := c.Client.Do(req.WithContext(ctx))
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if res.StatusCode == http.StatusOK { // graph is ready
			return extractGraph(res.Body)
		}
		if res.StatusCode != http.StatusNoContent {
			data, _ := ioutil.ReadAll(res.Body)
			return nil, fmt.Errorf("Received unexpected response while polling grapher %d: %s", res.StatusCode, data)
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("request time out reached after %d attempt(s): %s", attempts, ctx.Err().Error())
		case <-time.After(c.PollingInterval):
		}
	}
}

func extractGraph(r io.Reader) (io.ReadCloser, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return ioutil.NopCloser(bytes.NewReader(data)), nil
}

func newGraphRequest(endpoint *url.URL, method string, start, stop time.Time) (*http.Request, error) {
	u, _ := url.Parse(endpoint.String())
	q := u.Query()
	q.Set(queryStart, start.Format(time.RFC3339Nano))
	q.Set(queryStop, stop.Format(time.RFC3339Nano))
	u.RawQuery = q.Encode()
	return http.NewRequest(method, u.String(), nil)
}
