package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoginLimiterLocksAfterMaxFails(t *testing.T) {
	l := newLoginLimiter(3, time.Minute, time.Minute)
	now := time.Now()
	ip := "10.0.0.1"

	for i := 0; i < 3; i++ {
		if l.isLocked(ip, now) {
			t.Fatalf("should not be locked before reaching threshold, i=%d", i)
		}
		l.recordFailure(ip, now)
	}
	if !l.isLocked(ip, now) {
		t.Fatalf("expected lock after %d failures", 3)
	}
}

func TestLoginLimiterUnlocksAfterExpiry(t *testing.T) {
	l := newLoginLimiter(2, time.Minute, time.Minute)
	now := time.Now()
	ip := "10.0.0.2"

	l.recordFailure(ip, now)
	l.recordFailure(ip, now)
	if !l.isLocked(ip, now) {
		t.Fatalf("expected lock")
	}

	// 锁定时长过后应自动解锁并清空计数
	later := now.Add(time.Minute + time.Second)
	if l.isLocked(ip, later) {
		t.Fatalf("expected unlock after lock duration")
	}
}

func TestLoginLimiterWindowReset(t *testing.T) {
	l := newLoginLimiter(3, time.Minute, time.Minute)
	ip := "10.0.0.3"

	// 窗口内两次失败
	l.recordFailure(ip, time.Now())
	l.recordFailure(ip, time.Now())
	// 跨过窗口边界后应重新计数
	l.recordFailure(ip, time.Now().Add(time.Minute+time.Second))
	if l.isLocked(ip, time.Now().Add(time.Minute+time.Second)) {
		t.Fatalf("should not lock: window reset cleared prior failures")
	}
}

func TestLoginLimiterRecordSuccessClears(t *testing.T) {
	l := newLoginLimiter(2, time.Minute, time.Minute)
	ip := "10.0.0.4"

	l.recordFailure(ip, time.Now())
	l.recordSuccess(ip)
	l.recordFailure(ip, time.Now())
	if l.isLocked(ip, time.Now()) {
		t.Fatalf("success should have cleared prior failures")
	}
}

func TestLoginLimiterIsolatesByIP(t *testing.T) {
	l := newLoginLimiter(2, time.Minute, time.Minute)
	now := time.Now()

	l.recordFailure("1.1.1.1", now)
	l.recordFailure("1.1.1.1", now)
	if !l.isLocked("1.1.1.1", now) {
		t.Fatalf("expected 1.1.1.1 locked")
	}
	if l.isLocked("2.2.2.2", now) {
		t.Fatalf("2.2.2.2 must not be affected by 1.1.1.1")
	}
}

func TestIPFromRequest(t *testing.T) {
	tests := []struct {
		name   string
		xff    string
		remote string
		want   string
	}{
		{"direct", "", "203.0.113.5:1234", "203.0.113.5"},
		{"xff single", "198.51.100.7", "10.0.0.1:99", "198.51.100.7"},
		{"xff multi", "198.51.100.7, 10.0.0.1", "10.0.0.1:99", "198.51.100.7"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/login", nil)
			req.RemoteAddr = tc.remote
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			if got := ipFromRequest(req); got != tc.want {
				t.Fatalf("ipFromRequest = %q, want %q", got, tc.want)
			}
		})
	}
}
