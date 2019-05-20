package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/asecurityteam/vpcflow-diffd/pkg/domain"
	"github.com/asecurityteam/vpcflow-diffd/pkg/logs"
	"github.com/google/uuid"
)

var diffNamespace = uuid.NewSHA1(uuid.Nil, []byte("diff"))

// DiffHandler handles incoming HTTP requests for creating and retrieving new network graph diffs
type DiffHandler struct {
	LogProvider domain.LogFn
	Storage     domain.Storage
	Queuer      domain.Queuer
	Marker      domain.Marker
}

// Post creates a new diff
func (h *DiffHandler) Post(w http.ResponseWriter, r *http.Request) {
	logger := h.LogProvider(r.Context())
	diff, err := extractInput(r)
	if err != nil {
		logger.Info(logs.InvalidInput{Reason: err.Error()})
		writeJSONResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	exists, err := h.Storage.Exists(r.Context(), diff.ID)
	switch err.(type) {
	case nil:
	case domain.ErrInProgress:
		logger.Info(logs.Conflict{Reason: err.Error()})
		writeJSONResponse(w, http.StatusConflict, err.Error())
		return
	default:
		logger.Error(logs.DependencyFailure{Dependency: logs.DependencyStorage, Reason: err.Error()})
		writeJSONResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	// if data is returned, a diff already exists. return 409 and exit
	if exists {
		pStart := diff.PreviousStart.Format(time.RFC3339)
		pStop := diff.PreviousStop.Format(time.RFC3339)
		nStart := diff.NextStart.Format(time.RFC3339)
		nStop := diff.NextStop.Format(time.RFC3339)
		msg := fmt.Sprintf("diff for the time range %s to %s and time range %s to %s already exists", pStart, pStop, nStart, nStop)
		logger.Info(logs.Conflict{Reason: msg})
		writeJSONResponse(w, http.StatusConflict, msg)
		return
	}

	if err = h.Queuer.Queue(r.Context(), diff); err != nil {
		logger.Error(logs.DependencyFailure{Dependency: logs.DependencyQueuer, Reason: err.Error()})
		writeJSONResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// If mark fails, we don't fail the request since diff creation should be idempotent
	err = h.Marker.Mark(r.Context(), diff.ID)
	if err != nil {
		logger.Info(logs.DependencyFailure{Dependency: logs.DependencyMarker, Reason: err.Error()})
	}

	w.WriteHeader(http.StatusAccepted)
}

// Get retrieves a diff
func (h *DiffHandler) Get(w http.ResponseWriter, r *http.Request) {
	logger := h.LogProvider(r.Context())
	diff, err := extractInput(r)
	if err != nil {
		logger.Info(logs.InvalidInput{Reason: err.Error()})
		writeJSONResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	body, err := h.Storage.Get(r.Context(), diff.ID)
	switch err.(type) {
	case nil:
		defer body.Close()
	case domain.ErrInProgress:
		w.WriteHeader(http.StatusNoContent)
		return
	case domain.ErrNotFound:
		logger.Info(logs.NotFound{Reason: err.Error()})
		w.WriteHeader(http.StatusNotFound)
		return
	default:
		logger.Error(logs.DependencyFailure{Dependency: logs.DependencyStorage, Reason: err.Error()})
		writeJSONResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, body)
}

// extractInput attempts to extract the time range query parameters required by GET and POST.
// If any of the values are not valid RFC3339Nano or the input is invalid, an error is returned.
// Otherwise, the Diff domain type is returned with the "previous" and "next" time ranges set. A
// unique ID for the diff is also computed using these time values.
//
// Additionally, it truncates the time values to the nearest minute since anything with more
// precision doesn't really fit the vpc flow filter use case
func extractInput(r *http.Request) (domain.Diff, error) {
	pStart, pStop, err := validateTimeRange(r.URL.Query().Get("previous_start"), r.URL.Query().Get("previous_stop"))
	if err != nil {
		return domain.Diff{}, err
	}
	nStart, nStop, err := validateTimeRange(r.URL.Query().Get("next_start"), r.URL.Query().Get("next_stop"))
	if err != nil {
		return domain.Diff{}, err
	}
	if pStart.After(nStart) || pStop.After(nStop) {
		return domain.Diff{}, errors.New("the previous range should be before the next range")
	}
	name := pStart.String() + pStop.String() + nStart.String() + nStop.String()
	id := uuid.NewSHA1(diffNamespace, []byte(name)).String()
	return domain.Diff{
		ID:            id,
		PreviousStart: pStart.Truncate(time.Minute),
		PreviousStop:  pStop.Truncate(time.Minute),
		NextStart:     nStart.Truncate(time.Minute),
		NextStop:      nStop.Truncate(time.Minute),
	}, nil
}

// write the http response with the given status code and message
func writeJSONResponse(w http.ResponseWriter, statusCode int, message string) {
	msg := struct {
		Message string `json:"message"`
	}{
		Message: message,
	}
	w.Header().Add("Content-type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(msg)
}
