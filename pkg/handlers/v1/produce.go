package v1

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/asecurityteam/vpcflow-diffd/pkg/domain"
	"github.com/asecurityteam/vpcflow-diffd/pkg/logs"
)

type payload struct {
	ID            string `json:"id"`
	PreviousStart string `json:"previousStart"`
	PreviousStop  string `json:"previousStop"`
	NextStart     string `json:"nextStart"`
	NextStop      string `json:"nextStop"`
}

// Produce is a handler which performs the diff job, and stores the diff
type Produce struct {
	LogProvider domain.LoggerProvider
	Marker      domain.Marker
	Differ      domain.Differ
	Storage     domain.Storage
}

// ServeHTTP handles incoming HTTP requests, and creates a diff of the VPC network graphs given two time windows
func (h *Produce) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.LogProvider(r.Context())
	var body payload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		logger.Info(logs.InvalidInput{Reason: err.Error()})
		writeTextResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	diff, err := diffFromPayload(body)
	if err != nil {
		logger.Info(logs.InvalidInput{Reason: err.Error()})
		writeTextResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	dOut, err := h.Differ.Diff(r.Context(), diff)
	if err != nil {
		logger.Error(logs.DependencyFailure{Dependency: logs.DependencyDiffer, Reason: err.Error()})
		writeTextResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer dOut.Close()

	if err := h.Storage.Store(r.Context(), diff.ID, dOut); err != nil {
		logger.Error(logs.DependencyFailure{Dependency: logs.DependencyStorage, Reason: err.Error()})
		writeTextResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// We may want to improve this in the future to be a non-fatal error. Today if unmark fails,
	// fetching the diff will result in a perpetual "in progress" state. To mitigate this, we
	// report a failure to the caller signifying that the operation should be retried. This will
	// hopefully mitigate the amount of invalid state occurrence we may incur
	if err := h.Marker.Unmark(r.Context(), body.ID); err != nil {
		logger.Error(logs.DependencyFailure{Dependency: logs.DependencyMarker, Reason: err.Error()})
		writeTextResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func diffFromPayload(p payload) (domain.Diff, error) {
	if p.ID == "" {
		return domain.Diff{}, errors.New("missing ID field")
	}

	pStart, pStop, err := validateTimeRange(p.PreviousStart, p.PreviousStop)
	if err != nil {
		return domain.Diff{}, err
	}

	nStart, nStop, err := validateTimeRange(p.NextStart, p.NextStop)
	if err != nil {
		return domain.Diff{}, err
	}

	if pStart.After(nStart) || pStop.After(nStart) {
		return domain.Diff{}, errors.New("the previous range should be before the next range")
	}

	return domain.Diff{
		ID:            p.ID,
		PreviousStart: pStart,
		PreviousStop:  pStop,
		NextStart:     nStart,
		NextStop:      nStop,
	}, nil
}

func writeTextResponse(w http.ResponseWriter, statusCode int, msg string) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write([]byte(msg))
}
