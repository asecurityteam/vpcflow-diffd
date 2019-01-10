package v1

import (
	"errors"
	"time"
)

func validateTimeRange(start, end string) (time.Time, time.Time, error) {
	t1, err := time.Parse(time.RFC3339Nano, start)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	t2, err := time.Parse(time.RFC3339Nano, end)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if t1.After(t2) {
		return time.Time{}, time.Time{}, errors.New("start should be before stop")
	}
	return t1, t2, nil
}
