package server

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestNewAutodiscoveryRegistrationBuildsInstanceNameFromHostAndPort(t *testing.T) {
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("os.Hostname: %v", err)
	}

	for _, port := range []int{9001, 8080} {
		instanceName, _, err := newAutodiscoveryRegistration(port)
		if err != nil {
			t.Fatalf("newAutodiscoveryRegistration: %v", err)
		}

		want := hostname + "-" + strconv.Itoa(port)
		if instanceName != want {
			t.Errorf("instance name = %q, want %q", instanceName, want)
		}
	}
}

func TestNewAutodiscoveryRegistrationPublishesTheAgreedTXTRecords(t *testing.T) {
	_, text, err := newAutodiscoveryRegistration(9001)
	if err != nil {
		t.Fatalf("newAutodiscoveryRegistration: %v", err)
	}

	want := []string{"path=/", "api=1"}
	if len(text) != len(want) {
		t.Fatalf("got %d TXT records (%v), want %d", len(text), text, len(want))
	}

	for i, w := range want {
		if text[i] != w {
			t.Errorf("TXT[%d] = %q, want %q", i, text[i], w)
		}
	}
}

func TestNewAutodiscoveryRegistrationKeepsTXTRecordsAsKeyValuePairs(t *testing.T) {
	_, text, err := newAutodiscoveryRegistration(9001)
	if err != nil {
		t.Fatalf("newAutodiscoveryRegistration: %v", err)
	}

	for _, record := range text {
		key, value, ok := strings.Cut(record, "=")
		if !ok {
			t.Errorf("TXT record %q is not a key=value pair", record)
			continue
		}
		if key == "" || value == "" {
			t.Errorf("TXT record %q has an empty key or value", record)
		}
	}
}

func TestNewAutodiscoveryServerRegistersAndShutsDownCleanly(t *testing.T) {
	if testing.Short() {
		t.Skip("registration binds multicast sockets")
	}

	server, err := newAutodiscoveryServer(autodiscoveryServiceName, 9001)
	if err != nil {
		t.Fatalf("newAutodiscoveryServer: %v", err)
	}
	if server == nil {
		t.Fatal("newAutodiscoveryServer returned a nil server")
	}

	server.Shutdown()
}
