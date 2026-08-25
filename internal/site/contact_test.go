package site

import (
	"strings"
	"testing"
)

func TestFormatContactPushoverMessage(t *testing.T) {
	msg := formatContactPushoverMessage("Ada", "ada@example.com", "Neon", "Hello there")
	for _, part := range []string{"New website enquiry", "Ada", "ada@example.com", "Neon", "Hello there"} {
		if !strings.Contains(msg, part) {
			t.Fatalf("missing %q in %q", part, msg)
		}
	}

	long := strings.Repeat("x", 2000)
	truncated := formatContactPushoverMessage("Ada", "ada@example.com", "", long)
	if len(truncated) > pushoverMaxMessage {
		t.Fatalf("message too long: %d", len(truncated))
	}
	if !strings.HasSuffix(truncated, "...") {
		t.Fatalf("expected truncation marker")
	}
}

func TestNotifyContactEnquiryRequiresConfig(t *testing.T) {
	s := &Server{cfg: Config{}}
	err := s.NotifyContactEnquiry("Ada", "ada@example.com", "", "hi")
	if err == nil {
		t.Fatal("expected error when Pushover is unset")
	}
}
