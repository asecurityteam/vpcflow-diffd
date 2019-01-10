package queuer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"bitbucket.org/atlassian/vpcflow-diffd/pkg/domain"
)

type payload struct {
	ID            string `json:"id"`
	PreviousStart string `json:"previousStart"`
	PreviousStop  string `json:"previousStop"`
	NextStart     string `json:"nextStart"`
	NextStop      string `json:"nextStop"`
}

// DiffQueuer is a Queuer implementation which queues graph jobs onto a streaming appliance
type DiffQueuer struct {
	Endpoint *url.URL
	Client   *http.Client
}

// Queue enqueues a diff job onto a streaming appliance
func (q *DiffQueuer) Queue(ctx context.Context, diff domain.Diff) error {
	body := payload{
		ID:            diff.ID,
		PreviousStart: diff.PreviousStart.Format(time.RFC3339Nano),
		PreviousStop:  diff.PreviousStop.Format(time.RFC3339Nano),
		NextStart:     diff.NextStart.Format(time.RFC3339Nano),
		NextStop:      diff.NextStop.Format(time.RFC3339Nano),
	}
	rawBody, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, q.Endpoint.String(), bytes.NewReader(rawBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	res, err := q.Client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response from streaming appliance: %d", res.StatusCode)
	}
	return nil
}
