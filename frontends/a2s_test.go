package frontends

import (
	"bytes"
	"net/netip"
	"testing"

	"github.com/UselessMnemonic/proxygw/pkg/config"
)

func TestA2SInfoResponse(t *testing.T) {
	handler, err := NewA2SHandler("test", config.ProtocolUDP, netip.MustParseAddrPort("127.0.0.1:2457"), map[string]any{
		"name":        "Frost Hall",
		"map":         "Mistlands",
		"max_players": 10,
		"password":    true,
	})
	if err != nil {
		t.Fatalf("NewA2SHandler() error = %v", err)
	}

	a2s := handler.(*A2SHandler)
	response, query, ok := a2s.response([]byte(a2sInfoRequest))
	if !ok {
		t.Fatal("response() did not recognize A2S_INFO")
	}
	if query != a2sQueryInfo {
		t.Fatalf("response() query = %q, want %q", query, a2sQueryInfo)
	}

	if !bytes.HasPrefix(response, []byte(a2sHeader+"I\x11Frost Hall\x00Mistlands\x00valheim\x00Valheim\x00")) {
		t.Fatalf("response() prefix = % x", response)
	}
	if len(a2s.ShouldWarm()) != 0 {
		t.Fatal("A2S query should not queue a warm signal")
	}
}

func TestA2SChallengeResponse(t *testing.T) {
	handler, err := NewA2SHandler("test", config.ProtocolUDP, netip.MustParseAddrPort("127.0.0.1:2457"), map[string]any{
		"name": "Frost Hall",
		"map":  "Mistlands",
	})
	if err != nil {
		t.Fatalf("NewA2SHandler() error = %v", err)
	}

	response, query, ok := handler.(*A2SHandler).response([]byte(a2sHeader + "V\xff\xff\xff\xff"))
	if !ok {
		t.Fatal("response() did not recognize A2S_RULES")
	}
	if query != a2sQueryRules {
		t.Fatalf("response() query = %q, want %q", query, a2sQueryRules)
	}
	if !bytes.HasPrefix(response, []byte(a2sHeader+"A")) {
		t.Fatalf("response() = % x", response)
	}
}

func TestA2SRequiresNameAndMap(t *testing.T) {
	tests := []struct {
		name    string
		options map[string]any
	}{
		{
			name:    "missing name",
			options: map[string]any{"map": "Mistlands"},
		},
		{
			name:    "missing map",
			options: map[string]any{"name": "Frost Hall"},
		},
		{
			name:    "empty name",
			options: map[string]any{"name": "", "map": "Mistlands"},
		},
		{
			name:    "empty map",
			options: map[string]any{"name": "Frost Hall", "map": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewA2SHandler("test", config.ProtocolUDP, netip.MustParseAddrPort("127.0.0.1:2457"), tt.options)
			if err == nil {
				t.Fatal("NewA2SHandler() error = nil, want error")
			}
		})
	}
}
