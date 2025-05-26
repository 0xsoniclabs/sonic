package metrics

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/stretchr/testify/require"
)

func TestMeasureDbDir_LogsDBDirSizeEveryMinute(t *testing.T) {
	// create a temporary directory for testing
	tempDir := t.TempDir()

	// create some data to write to the test file
	testData := []byte("test data")
	lenTestData := len(testData)

	// create a test file in the temporary directory
	f := createTestFile(t, tempDir, "file1")

	gaugeName := "testGauge"

	ctx, cancel := context.WithCancel(t.Context())
	go measureDbDir(ctx, gaugeName, tempDir, time.Millisecond)

	for i := range 5 {
		writeTestData(t, f, testData)
		time.Sleep(time.Millisecond * 2)
		snapshotValueEquals(t, gaugeName, int64(lenTestData*(i+1)))
	}
	// we need to cancel the context to stop measureDbDir
	cancel()
}

func TestMeasureDbDir_LoopCanBeCancelled(t *testing.T) {
	tempDir := t.TempDir()
	done := make(chan struct{})

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		defer close(done)
		measureDbDir(ctx, "testGauge", tempDir, time.Millisecond)
	}()
	cancel()

	select {
	case <-done: // Success: the goroutine exited
	case <-time.After(time.Millisecond * 3):
		t.Fatal("timeout: measureDbDir did not stop after cancellation")
	}
}

func TestSetDataDir_StartsBackgroundProcessesOnlyOnce(t *testing.T) {
	before := runtime.NumGoroutine()
	SetDataDir(t.TempDir())
	SetDataDir(t.TempDir())
	after := runtime.NumGoroutine()
	if after-before != 2 {
		t.Fatalf("expected 2 goroutines to be started, got %d", after-before)
	}
}

// createTestFile is a helper functions that creates a file. It should be closed
// in a defer after the call to this function.
func createTestFile(t *testing.T, dir string, name string) *os.File {
	testFile := filepath.Join(dir, name)
	f, err := os.OpenFile(testFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	require.NoError(t, err)
	return f
}

// writeTestData is a helper function that writes data to a file.
func writeTestData(t *testing.T, f *os.File, data []byte) {
	_, err := f.Write(data)
	require.NoError(t, err)
}

// snapshotValueEquals is a helper function that gets the gauge with the given
// name and checks if the gauge value equals the expected value.
func snapshotValueEquals(t *testing.T, gaugeName string, expectedSize int64) {
	gauge := metrics.GetOrRegisterGauge(gaugeName, nil)
	require.Equal(t, expectedSize, gauge.Snapshot().Value())
}
