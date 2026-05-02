package frontends

import (
	"net/netip"
	"testing"

	"github.com/UselessMnemonic/proxygw/pkg/config"
)

func TestA2SRequiresCoreOptions(t *testing.T) {
	tests := []struct {
		name    string
		options map[string]any
	}{
		{
			name:    "missing name",
			options: map[string]any{"version": "0.0.0"},
		},
		{
			name:    "missing version",
			options: map[string]any{"name": "Frost Hall"},
		},
		{
			name:    "empty name",
			options: map[string]any{"name": "", "version": "0.0.0"},
		},
		{
			name:    "empty version",
			options: map[string]any{"name": "Frost Hall", "version": ""},
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

func TestA2SKeywordsDefaultFromVersion(t *testing.T) {
	info, err := newA2SInfo(netip.MustParseAddrPort("127.0.0.1:2457"), map[string]any{
		"name":    "Frost Hall",
		"version": "0.222.0",
	})
	if err != nil {
		t.Fatalf("newA2SInfo() error = %v", err)
	}

	if info.Keywords != "g=0.222.0,n=36,m=" {
		t.Fatalf("Keywords = %q, want %q", info.Keywords, "g=0.222.0,n=36,m=")
	}
}

func TestA2SIgnoresUnknownOptions(t *testing.T) {
	info, err := newA2SInfo(netip.MustParseAddrPort("127.0.0.1:2457"), map[string]any{
		"name":     "Frost Hall",
		"version":  "0.222.0",
		"keywords": "custom",
	})
	if err != nil {
		t.Fatalf("newA2SInfo() error = %v", err)
	}

	if info.Keywords != "g=0.222.0,n=36,m=" {
		t.Fatalf("Keywords = %q, want %q", info.Keywords, "g=0.222.0,n=36,m=")
	}
}
