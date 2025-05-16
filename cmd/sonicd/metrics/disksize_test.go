package metrics

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/stretchr/testify/require"
)

func TestMeasureDbDir(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create some test files in the temporary directory
	testFile1 := filepath.Join(tempDir, "file1")
	testFile2 := filepath.Join(tempDir, "file2")
	err := os.WriteFile(testFile1, []byte("test data"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, []byte("more test data"), 0644)
	require.NoError(t, err)

	gaugeName := "test_db_size"

	// Run measureDbDir in a goroutine and stop it after a short duration
	go func() {
		measureDbDir(gaugeName, tempDir)
	}()
	// disk gets measured once per minute, so we have to wait for that
	time.Sleep(1*time.Second + time.Minute)

	// Verify the gauge value matches the total size of the files
	expectedSize := int64(len("test data") + len("more test data"))
	gauge := metrics.GetOrRegisterGauge(gaugeName, nil)
	require.Equal(t, expectedSize, gauge.Snapshot().Value())
}
