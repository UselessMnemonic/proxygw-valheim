package a2s

import (
	"bytes"
	"encoding/binary"
	"testing"
)

const berkeleyGamesInfoPacket = "" +
	"\xff\xff\xff\xffI\x11" +
	"Berkeley Games\x00" +
	"Berkeley Games\x00" +
	"valheim\x00" +
	"\x00" +
	"\x00\x00" +
	"\x00\x0a\x00" +
	"dl" +
	"\x01\x00" +
	"1.0.0.0\x00" +
	"\xb1" +
	"\x98\x09" +
	"\x0c\x28\xc4\xbf\xf2\xc1\x40\x01" +
	"g=0.221.12,n=36,m=\x00" +
	"\x2a\xa0\x0d\x00\x00\x00\x00\x00"

func TestInfoBerkeleyGamesParseUnparse(t *testing.T) {
	packet := []byte(berkeleyGamesInfoPacket)
	info, err := ParseInfo(packet)
	if err != nil {
		t.Fatalf("ParseInfo() error = %v", err)
	}
	if info.EDF != EDFPort|EDFSteamID|EDFKeywords|EDFGameID {
		t.Fatalf("EDF = %#x, want %#x", info.EDF, EDFPort|EDFSteamID|EDFKeywords|EDFGameID)
	}

	got, err := info.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}
	if !bytes.Equal(got, packet) {
		t.Fatalf("parse/unparse mismatch\ngot  = % x\nwant = % x", got, packet)
	}
}

func TestInfoMarshalUsesPacketEDF(t *testing.T) {
	info := Info{
		Name:        "No Extras",
		Map:         "No Extras",
		Folder:      "valheim",
		AppID:       0,
		ServerType:  ServerTypeDedicated,
		Environment: EnvironmentLinux,
		Visibility:  VisibilityPrivate,
		Version:     "1.0.0.0",
		EDF:         0,
		Port:        2456,
		SteamID:     SteamID(UniversePublic, AccountTypeAnonGameServer, 49650, 3217303564),
		Keywords:    "g=0.221.12,n=36,m=",
		GameID:      892970,
	}

	packet, err := info.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}

	if !bytes.HasSuffix(packet, []byte("1.0.0.0\x00\x00")) {
		t.Fatalf("packet = % x, want zero EDF with no trailing optional fields", packet)
	}
}

func TestInfoParseWithoutEDF(t *testing.T) {
	packet := []byte(Header + "I\x11" +
		"No Extras\x00" +
		"No Extras\x00" +
		"valheim\x00" +
		"\x00" +
		"\x00\x00" +
		"\x00\x0a\x00" +
		"dl" +
		"\x01\x00" +
		"1.0.0.0\x00")

	info, err := ParseInfo(packet)
	if err != nil {
		t.Fatalf("ParseInfo() error = %v", err)
	}
	if info.EDF != 0 {
		t.Fatalf("EDF = %#x, want 0", info.EDF)
	}
}

func TestSteamIDBuildsBerkeleyGamesID(t *testing.T) {
	id := SteamID(UniversePublic, AccountTypeAnonGameServer, 49650, 3217303564)
	if id != 0x0140c1f2bfc4280c {
		t.Fatalf("SteamID() = %#x, want %#x", id, uint64(0x0140c1f2bfc4280c))
	}

	var got [8]byte
	binary.LittleEndian.PutUint64(got[:], id)
	want := []byte{0x0c, 0x28, 0xc4, 0xbf, 0xf2, 0xc1, 0x40, 0x01}
	if !bytes.Equal(got[:], want) {
		t.Fatalf("SteamID() bytes = % x, want % x", got, want)
	}
}
