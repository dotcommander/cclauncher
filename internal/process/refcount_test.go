package process

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// createTestManager creates a RefCountManager with temporary paths for testing
func createTestManager(t testing.TB) *RefCountManager {
	tempDir := t.TempDir()
	refFile := filepath.Join(tempDir, ".ccg.refcount")
	return NewRefCountManagerWithPaths(tempDir, refFile)
}

func TestRefCountManager_BasicOperations(t *testing.T) {
	mgr := createTestManager(t)

	// Initial count should be 0
	count, err := mgr.GetRefCount()
	if err != nil {
		t.Fatalf("Failed to get initial count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected initial count 0, got %d", count)
	}

	// Increment
	if incErr := mgr.IncrementRefCount(); incErr != nil {
		t.Fatalf("Failed to increment: %v", incErr)
	}

	count, err = mgr.GetRefCount()
	if err != nil {
		t.Fatalf("Failed to get count after increment: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1 after increment, got %d", count)
	}

	// Decrement
	count, err = mgr.DecrementRefCount()
	if err != nil {
		t.Fatalf("Failed to decrement: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0 after decrement, got %d", count)
	}
}

func TestRefCountManager_MultipleIncrements(t *testing.T) {
	mgr := createTestManager(t)

	// Simulate 5 concurrent sessions
	for i := 1; i <= 5; i++ {
		if err := mgr.IncrementRefCount(); err != nil {
			t.Fatalf("Failed to increment at step %d: %v", i, err)
		}
	}

	count, err := mgr.GetRefCount()
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected count 5, got %d", count)
	}

	// Decrement one session at a time
	for i := 4; i >= 0; i-- {
		count, err := mgr.DecrementRefCount()
		if err != nil {
			t.Fatalf("Failed to decrement at step %d: %v", i, err)
		}
		if count != i {
			t.Errorf("Expected count %d after decrement, got %d", i, count)
		}
	}
}

func TestRefCountManager_DecrementBelowZero(t *testing.T) {
	mgr := createTestManager(t)

	// Decrement when count is 0 (should not go below 0)
	count, err := mgr.DecrementRefCount()
	if err != nil {
		t.Fatalf("Failed to decrement: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count to stay at 0, got %d", count)
	}

	// Verify file still shows 0
	count, err = mgr.GetRefCount()
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}

func TestRefCountManager_ResetRefCount(t *testing.T) {
	mgr := createTestManager(t)

	// Set count to 10
	for i := 0; i < 10; i++ {
		_ = mgr.IncrementRefCount()
	}

	count, _ := mgr.GetRefCount()
	if count != 10 {
		t.Errorf("Expected count 10, got %d", count)
	}

	// Reset
	if err := mgr.ResetRefCount(); err != nil {
		t.Fatalf("Failed to reset: %v", err)
	}

	count, err := mgr.GetRefCount()
	if err != nil {
		t.Fatalf("Failed to get count after reset: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0 after reset, got %d", count)
	}
}

func TestRefCountManager_ConcurrentAccess(t *testing.T) {
	mgr := createTestManager(t)
	var wg sync.WaitGroup

	// Simulate 50 concurrent increments
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = mgr.IncrementRefCount()
		}()
	}

	wg.Wait()

	count, err := mgr.GetRefCount()
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	if count != 50 {
		t.Errorf("Expected count 50 after concurrent increments, got %d", count)
	}

	// Simulate 50 concurrent decrements
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = mgr.DecrementRefCount()
		}()
	}

	wg.Wait()

	count, err = mgr.GetRefCount()
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0 after concurrent decrements, got %d", count)
	}
}

func TestRefCountManager_PersistenceAcrossInstances(t *testing.T) {
	tempDir := t.TempDir()
	refFile := filepath.Join(tempDir, ".ccg.refcount")

	// First manager instance
	mgr1 := NewRefCountManagerWithPaths(tempDir, refFile)
	for i := 0; i < 3; i++ {
		_ = mgr1.IncrementRefCount()
	}

	// Second manager instance (simulates new CLI invocation)
	mgr2 := NewRefCountManagerWithPaths(tempDir, refFile)
	count, err := mgr2.GetRefCount()
	if err != nil {
		t.Fatalf("Failed to get count from second instance: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected count 3 to persist across instances, got %d", count)
	}

	// Increment and decrement from different instances
	_ = mgr2.IncrementRefCount() // Count should be 4
	_, _ = mgr1.DecrementRefCount() // Count should be 3

	count, _ = mgr2.GetRefCount()
	if count != 3 {
		t.Errorf("Expected count 3 after mixed operations, got %d", count)
	}
}

func TestRefCountManager_InvalidFileContent(t *testing.T) {
	tempDir := t.TempDir()
	refFile := filepath.Join(tempDir, ".ccg.refcount")

	// Write invalid content to refcount file
	_ = os.MkdirAll(tempDir, 0755)
	_ = os.WriteFile(refFile, []byte("invalid"), 0600)

	mgr := NewRefCountManagerWithPaths(tempDir, refFile)

	// Should return 0 for invalid content
	count, err := mgr.GetRefCount()
	if err != nil {
		t.Fatalf("Expected no error for invalid content, got: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0 for invalid content, got %d", count)
	}
}

func TestGetRefCountFile(t *testing.T) {
	path := GetRefCountFile()
	if path == "" {
		t.Error("Expected non-empty refcount file path")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("Expected absolute path, got: %s", path)
	}
}

func TestRefCountManager_RefCountFile(t *testing.T) {
	tempDir := t.TempDir()
	refFile := filepath.Join(tempDir, ".ccg.refcount")
	mgr := NewRefCountManagerWithPaths(tempDir, refFile)

	path := mgr.RefCountFile()
	if path != refFile {
		t.Errorf("Expected %s, got %s", refFile, path)
	}
}

func BenchmarkRefCountManager_Increment(b *testing.B) {
	mgr := createTestManager(b)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = mgr.IncrementRefCount()
	}
}

func BenchmarkRefCountManager_Decrement(b *testing.B) {
	mgr := createTestManager(b)

	// Pre-populate
	for i := 0; i < b.N; i++ {
		_ = mgr.IncrementRefCount()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = mgr.DecrementRefCount()
	}
}

func BenchmarkRefCountManager_Get(b *testing.B) {
	mgr := createTestManager(b)
	_ = mgr.IncrementRefCount()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = mgr.GetRefCount()
	}
}
