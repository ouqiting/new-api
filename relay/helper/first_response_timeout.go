package helper

import (
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/types"
)

// ErrFirstResponseTimeout is returned when an upstream response produces no
// first byte within the configured per-channel first-response timeout.
var ErrFirstResponseTimeout = errors.New("first response timeout")

// NewFirstResponseTimeoutError builds a retryable error (HTTP 500) signalling
// that the upstream produced no first response within the configured timeout.
func NewFirstResponseTimeoutError() *types.NewAPIError {
	return types.NewOpenAIError(ErrFirstResponseTimeout, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
}

// WrapFirstResponseTimeout returns r normally when timeout <= 0, otherwise
// returns an io.ReadCloser that fails with ErrFirstResponseTimeout if the first
// Read does not deliver any data within the given timeout. Subsequent reads are
// passed through untouched (only the very first byte is bounded).
func WrapFirstResponseTimeout(r io.ReadCloser, timeout time.Duration) io.ReadCloser {
	if r == nil || timeout <= 0 {
		return r
	}
	return &firstResponseTimeoutReader{
		r:       r,
		timeout: timeout,
		first:   1,
	}
}

type firstResponseTimeoutReader struct {
	r       io.ReadCloser
	timeout time.Duration
	first   int32
}

func (f *firstResponseTimeoutReader) Close() error {
	if f.r == nil {
		return nil
	}
	return f.r.Close()
}

type readResult struct {
	n   int
	err error
}

func (f *firstResponseTimeoutReader) Read(p []byte) (int, error) {
	if atomic.CompareAndSwapInt32(&f.first, 1, 0) {
		return f.readFirst(p)
	}
	return f.r.Read(p)
}

func (f *firstResponseTimeoutReader) readFirst(p []byte) (int, error) {
	result := make(chan readResult, 1)
	go func() {
		n, err := f.r.Read(p)
		result <- readResult{n: n, err: err}
	}()
	timer := time.NewTimer(f.timeout)
	defer timer.Stop()
	select {
	case r := <-result:
		return r.n, r.err
	case <-timer.C:
		return 0, ErrFirstResponseTimeout
	}
}
