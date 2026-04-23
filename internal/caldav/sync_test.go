package caldav

import (
	"errors"
	"strings"
	"testing"
)

func TestSyncSource_ConcurrentCallsReturnErrSyncInProgress(t *testing.T) {
	// Acquire the per-source lock manually to simulate an in-flight sync.
	mu := lockFor(999)
	mu.Lock()
	defer mu.Unlock()

	src := CalendarSource{
		ID:   999,
		Name: "concurrent-test",
		URL:  "https://example.invalid/",
	}

	err := SyncSource(nil, src, "secret")
	if !errors.Is(err, ErrSyncInProgress) {
		t.Fatalf("expected ErrSyncInProgress, got %v", err)
	}
}

func TestSyncSource_PrivateIPRejected(t *testing.T) {
	// Fresh source ID so the lock is free and the URL check runs.
	src := CalendarSource{
		ID:   1001,
		Name: "ssrf-test",
		URL:  "http://127.0.0.1/",
	}

	err := SyncSource(nil, src, "secret")
	if err == nil {
		t.Fatal("expected SSRF rejection, got nil")
	}
	if !strings.Contains(err.Error(), "rejected") {
		t.Errorf("expected rejection error, got: %v", err)
	}
}

func TestSyncSource_MetadataEndpointRejected(t *testing.T) {
	src := CalendarSource{
		ID:   1002,
		Name: "aws-metadata",
		URL:  "https://169.254.169.254/latest/meta-data/",
	}
	err := SyncSource(nil, src, "secret")
	if err == nil {
		t.Fatal("expected SSRF rejection, got nil")
	}
}
