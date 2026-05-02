package frontends

import "testing"

func TestLooksLikeValheimGameplay(t *testing.T) {
	tests := []struct {
		name   string
		packet []byte
		want   bool
	}{
		{
			name:   "steam networking challenge request",
			packet: []byte{steamNetworkingUDPChallengeRequest, 0x0d, 0x01, 0x00, 0x00, 0x00},
			want:   true,
		},
		{
			name:   "steam networking connect request",
			packet: []byte{steamNetworkingUDPConnectRequest, 0x0d, 0x01, 0x00, 0x00, 0x00},
			want:   true,
		},
		{
			name:   "a2s info on wrong port",
			packet: []byte(a2sInfoRequest),
			want:   false,
		},
		{
			name:   "single byte message id",
			packet: []byte{steamNetworkingUDPConnectRequest},
			want:   false,
		},
		{
			name:   "random udp",
			packet: []byte("hello there"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeValheimGameplay(tt.packet); got != tt.want {
				t.Fatalf("looksLikeValheimGameplay() = %t, want %t", got, tt.want)
			}
		})
	}
}
