package service

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchedulingLogServiceKeepsNewestEventsWithinCapacity(t *testing.T) {
	svc := NewSchedulingLogService(3)

	for i := 1; i <= 5; i++ {
		svc.Record(SchedulingLogEvent{AccountID: int64(i), Reason: fmt.Sprintf("event-%d", i)})
	}

	logs := svc.List(10)
	require.Len(t, logs, 3)
	require.Equal(t, int64(5), logs[0].AccountID)
	require.Equal(t, int64(4), logs[1].AccountID)
	require.Equal(t, int64(3), logs[2].AccountID)
}

func TestSchedulingLogServiceIsConcurrentSafe(t *testing.T) {
	svc := NewSchedulingLogService(50)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			svc.Record(SchedulingLogEvent{AccountID: int64(i), Reason: "selected"})
		}(i)
	}
	wg.Wait()

	logs := svc.List(100)
	require.Len(t, logs, 50)
	for _, item := range logs {
		require.NotZero(t, item.CreatedAt)
		require.Equal(t, "selected", item.Reason)
	}
}
