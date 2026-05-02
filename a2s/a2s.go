package a2s

import (
	"bytes"
	"fmt"
)

const (
	ReadBufferSize = 1400
	Header         = "\xff\xff\xff\xff"
	InfoRequest    = "\xff\xff\xff\xffTSource Engine Query\x00"
	InfoVersion    = "1.0.0.0"
)

type Query string

const (
	QueryInfo      Query = "a2s_info"
	QueryPlayer    Query = "a2s_player"
	QueryRules     Query = "a2s_rules"
	QueryChallenge Query = "a2s_challenge"
	QueryPing      Query = "a2a_ping"
)

func writeString(b *bytes.Buffer, value string) {
	b.WriteString(value)
	b.WriteByte(0)
}

func parseString(packet []byte) (string, []byte, error) {
	idx := bytes.IndexByte(packet, 0)
	if idx < 0 {
		return "", nil, fmt.Errorf("missing null terminator")
	}
	return string(packet[:idx]), packet[idx+1:], nil
}
