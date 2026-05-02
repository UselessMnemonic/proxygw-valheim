package a2s

import (
	"bytes"
	"encoding/binary"
)

func ChallengeResponse(challenge uint32) []byte {
	var b bytes.Buffer
	b.WriteString(Header)
	b.WriteByte('A')
	_ = binary.Write(&b, binary.LittleEndian, int32(challenge))
	return b.Bytes()
}

func PlayerResponse() []byte {
	var b bytes.Buffer
	b.WriteString(Header)
	b.WriteByte('D')
	b.WriteByte(0)
	return b.Bytes()
}

func RulesResponse() []byte {
	var b bytes.Buffer
	b.WriteString(Header)
	b.WriteByte('E')
	_ = binary.Write(&b, binary.LittleEndian, uint16(0))
	return b.Bytes()
}

func PingResponse() []byte {
	return []byte(Header + "j00000000000000\x00")
}

func HasChallenge(packet []byte, challenge uint32) bool {
	if len(packet) < len(Header)+5 {
		return false
	}
	return binary.LittleEndian.Uint32(packet[len(Header)+1:]) == challenge
}

func InfoHasChallenge(packet []byte, challenge uint32) bool {
	if len(packet) < len(InfoRequest)+4 {
		return false
	}
	return binary.LittleEndian.Uint32(packet[len(InfoRequest):]) == challenge
}
