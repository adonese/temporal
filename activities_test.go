package iplocate

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestIPActivities_GetTimeZone(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	a := &IPActivities{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}

	ip := "1.1.1.1" // random IP address
	tz, err := a.GetTimeZone(context.Background(), ip)
	if err != nil {
		t.Fatalf("GetTimeZone failed: %v", err)
	}

	if tz == "" {
		t.Error("expected non-empty timezone")
	}

	t.Logf("IP: %s, TimeZone: %s", ip, tz)
}
