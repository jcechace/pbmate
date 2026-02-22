package sdk

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEntity is a minimal type for testing waitForTerminal.
type testEntity struct {
	status Status
	errMsg string
}

func makeWaitParams(calls *[]testEntity, entities ...testEntity) waitParams[*testEntity] {
	idx := 0
	return waitParams[*testEntity]{
		get: func(_ context.Context, _ string) (*testEntity, error) {
			if idx >= len(entities) {
				// Stay on the last entity if we over-poll.
				e := entities[len(entities)-1]
				*calls = append(*calls, e)
				return &e, nil
			}
			e := entities[idx]
			idx++
			*calls = append(*calls, e)
			return &e, nil
		},
		status: func(e *testEntity) Status { return e.status },
		errMsg: func(e *testEntity) string { return e.errMsg },
		log:    slog.New(slog.DiscardHandler),
		entity: "test",
	}
}

func TestWaitForTerminal_ImmediateDone(t *testing.T) {
	var calls []testEntity
	p := makeWaitParams(&calls, testEntity{status: StatusDone})

	result, err := waitForTerminal(context.Background(), "test-1", time.Millisecond, p)
	require.NoError(t, err)
	assert.Equal(t, StatusDone, result.status)
	assert.Len(t, calls, 1, "should poll exactly once for immediate terminal")
}

func TestWaitForTerminal_PollsThroughNonTerminal(t *testing.T) {
	var calls []testEntity
	p := makeWaitParams(&calls,
		testEntity{status: StatusStarting},
		testEntity{status: StatusRunning},
		testEntity{status: StatusDone},
	)

	result, err := waitForTerminal(context.Background(), "test-1", time.Millisecond, p)
	require.NoError(t, err)
	assert.Equal(t, StatusDone, result.status)
	assert.Len(t, calls, 3)
}

func TestWaitForTerminal_ErrorStatus(t *testing.T) {
	var calls []testEntity
	p := makeWaitParams(&calls,
		testEntity{status: StatusRunning},
		testEntity{status: StatusError, errMsg: "something went wrong"},
	)

	result, err := waitForTerminal(context.Background(), "test-1", time.Millisecond, p)
	require.Error(t, err)

	var opErr *OperationError
	require.True(t, errors.As(err, &opErr))
	assert.Equal(t, "test-1", opErr.Name)
	assert.Equal(t, "something went wrong", opErr.Message)
	assert.Equal(t, StatusError, result.status, "entity should still be returned on error")
}

func TestWaitForTerminal_PartlyDoneStatus(t *testing.T) {
	var calls []testEntity
	p := makeWaitParams(&calls,
		testEntity{status: StatusPartlyDone, errMsg: "partial failure"},
	)

	result, err := waitForTerminal(context.Background(), "test-1", time.Millisecond, p)
	require.Error(t, err)

	var opErr *OperationError
	require.True(t, errors.As(err, &opErr))
	assert.Equal(t, "partial failure", opErr.Message)
	assert.Equal(t, StatusPartlyDone, result.status)
}

func TestWaitForTerminal_NotFoundRetry(t *testing.T) {
	// Simulate entity not found on first call, then appearing.
	idx := 0
	var calls []string
	p := waitParams[*testEntity]{
		get: func(_ context.Context, name string) (*testEntity, error) {
			idx++
			if idx <= 2 {
				calls = append(calls, "not-found")
				return nil, ErrNotFound
			}
			calls = append(calls, "found")
			return &testEntity{status: StatusDone}, nil
		},
		status: func(e *testEntity) Status { return e.status },
		errMsg: func(e *testEntity) string { return e.errMsg },
		log:    slog.New(slog.DiscardHandler),
		entity: "test",
	}

	result, err := waitForTerminal(context.Background(), "test-1", time.Millisecond, p)
	require.NoError(t, err)
	assert.Equal(t, StatusDone, result.status)
	assert.Equal(t, []string{"not-found", "not-found", "found"}, calls)
}

func TestWaitForTerminal_NonNotFoundError(t *testing.T) {
	someErr := errors.New("connection lost")
	p := waitParams[*testEntity]{
		get: func(_ context.Context, _ string) (*testEntity, error) {
			return nil, someErr
		},
		status: func(e *testEntity) Status { return e.status },
		errMsg: func(e *testEntity) string { return e.errMsg },
		log:    slog.New(slog.DiscardHandler),
		entity: "test",
	}

	_, err := waitForTerminal(context.Background(), "test-1", time.Millisecond, p)
	require.Error(t, err)
	assert.ErrorIs(t, err, someErr)
	assert.Contains(t, err.Error(), "wait for test \"test-1\"")
}

func TestWaitForTerminal_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	p := waitParams[*testEntity]{
		get: func(_ context.Context, _ string) (*testEntity, error) {
			callCount++
			if callCount >= 2 {
				cancel()
			}
			return &testEntity{status: StatusRunning}, nil
		},
		status: func(e *testEntity) Status { return e.status },
		errMsg: func(e *testEntity) string { return e.errMsg },
		log:    slog.New(slog.DiscardHandler),
		entity: "test",
	}

	result, err := waitForTerminal(ctx, "test-1", time.Millisecond, p)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, StatusRunning, result.status, "last observed entity should be returned")
}

func TestWaitForTerminal_OnProgressCallback(t *testing.T) {
	var progressCalls []*testEntity
	var calls []testEntity
	p := makeWaitParams(&calls,
		testEntity{status: StatusStarting},
		testEntity{status: StatusRunning},
		testEntity{status: StatusDone},
	)
	p.onProgress = func(e *testEntity) {
		progressCalls = append(progressCalls, e)
	}

	_, err := waitForTerminal(context.Background(), "test-1", time.Millisecond, p)
	require.NoError(t, err)
	require.Len(t, progressCalls, 3, "onProgress called for every poll including terminal")
	assert.Equal(t, StatusStarting, progressCalls[0].status)
	assert.Equal(t, StatusRunning, progressCalls[1].status)
	assert.Equal(t, StatusDone, progressCalls[2].status)
}

func TestWaitForTerminal_DefaultPollInterval(t *testing.T) {
	// Passing 0 interval should use defaultPollInterval (1s).
	// We just verify it doesn't panic and works. We use a real terminal
	// status on the first call to avoid a long test.
	var calls []testEntity
	p := makeWaitParams(&calls, testEntity{status: StatusDone})

	result, err := waitForTerminal(context.Background(), "test-1", 0, p)
	require.NoError(t, err)
	assert.Equal(t, StatusDone, result.status)
}

func TestWaitForTerminal_CancelledStatus(t *testing.T) {
	var calls []testEntity
	p := makeWaitParams(&calls, testEntity{status: StatusCancelled})

	result, err := waitForTerminal(context.Background(), "test-1", time.Millisecond, p)
	require.NoError(t, err, "cancelled is terminal but not an error")
	assert.Equal(t, StatusCancelled, result.status)
}
