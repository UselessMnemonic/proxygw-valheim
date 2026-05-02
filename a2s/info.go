package a2s

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type ServerType byte

const (
	ServerTypeDedicated    ServerType = 'd'
	ServerTypeNonDedicated ServerType = 'l'
	ServerTypeProxy        ServerType = 'p'
)

type Environment byte

const (
	EnvironmentLinux   Environment = 'l'
	EnvironmentWindows Environment = 'w'
	EnvironmentMac     Environment = 'm'
	EnvironmentMacAlt  Environment = 'o'
)

type Visibility byte

const (
	VisibilityPublic  Visibility = 0
	VisibilityPrivate Visibility = 1
)

type VAC byte

const (
	VACUnsecured VAC = 0
	VACSecured   VAC = 1
)

type Info struct {
	Name        string
	Map         string
	Folder      string
	Game        string
	AppID       uint16
	Players     uint8
	MaxPlayers  uint8
	Bots        uint8
	ServerType  ServerType
	Environment Environment
	Visibility  Visibility
	VAC         VAC
	Version     string
	EDF         EDF
	Port        uint16
	Keywords    string
	SteamID     uint64
	GameID      uint64
}

func (i Info) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(Header)
	b.WriteByte('I')
	b.WriteByte(17)
	writeString(&b, i.Name)
	writeString(&b, i.Map)
	writeString(&b, i.Folder)
	writeString(&b, i.Game)
	_ = binary.Write(&b, binary.LittleEndian, i.AppID)
	b.WriteByte(i.Players)
	b.WriteByte(i.MaxPlayers)
	b.WriteByte(i.Bots)
	b.WriteByte(byte(i.ServerType))
	b.WriteByte(byte(i.Environment))
	b.WriteByte(byte(i.Visibility))
	b.WriteByte(byte(i.VAC))
	writeString(&b, i.Version)
	b.WriteByte(byte(i.EDF))
	if i.EDF&EDFPort != 0 {
		_ = binary.Write(&b, binary.LittleEndian, i.Port)
	}
	if i.EDF&EDFSteamID != 0 {
		_ = binary.Write(&b, binary.LittleEndian, i.SteamID)
	}
	if i.EDF&EDFKeywords != 0 {
		writeString(&b, i.Keywords)
	}
	if i.EDF&EDFGameID != 0 {
		_ = binary.Write(&b, binary.LittleEndian, i.GameID)
	}
	return b.Bytes(), nil
}

func (i *Info) UnmarshalBinary(packet []byte) error {
	info, err := ParseInfo(packet)
	if err != nil {
		return err
	}
	*i = info
	return nil
}

func ParseInfo(packet []byte) (Info, error) {
	var info Info
	if len(packet) < len(Header)+2 {
		return info, fmt.Errorf("packet too short")
	}
	if !bytes.HasPrefix(packet, []byte(Header+"I\x11")) {
		return info, fmt.Errorf("not an A2S_INFO response")
	}

	rest := packet[len(Header)+2:]
	var err error
	if info.Name, rest, err = parseString(rest); err != nil {
		return info, fmt.Errorf("name: %w", err)
	}
	if info.Map, rest, err = parseString(rest); err != nil {
		return info, fmt.Errorf("map: %w", err)
	}
	if info.Folder, rest, err = parseString(rest); err != nil {
		return info, fmt.Errorf("folder: %w", err)
	}
	if info.Game, rest, err = parseString(rest); err != nil {
		return info, fmt.Errorf("game: %w", err)
	}
	if len(rest) < 9 {
		return info, fmt.Errorf("fixed fields too short")
	}

	info.AppID = binary.LittleEndian.Uint16(rest)
	info.Players = rest[2]
	info.MaxPlayers = rest[3]
	info.Bots = rest[4]
	info.ServerType = ServerType(rest[5])
	info.Environment = Environment(rest[6])
	info.Visibility = Visibility(rest[7])
	info.VAC = VAC(rest[8])
	rest = rest[9:]

	if info.Version, rest, err = parseString(rest); err != nil {
		return info, fmt.Errorf("version: %w", err)
	}
	if len(rest) == 0 {
		return info, nil
	}

	info.EDF = EDF(rest[0])
	rest = rest[1:]
	if info.EDF&EDFPort != 0 {
		if len(rest) < 2 {
			return info, fmt.Errorf("missing port")
		}
		info.Port = binary.LittleEndian.Uint16(rest)
		rest = rest[2:]
	}
	if info.EDF&EDFSteamID != 0 {
		if len(rest) < 8 {
			return info, fmt.Errorf("missing steam id")
		}
		info.SteamID = binary.LittleEndian.Uint64(rest)
		rest = rest[8:]
	}
	if info.EDF&EDFKeywords != 0 {
		if info.Keywords, rest, err = parseString(rest); err != nil {
			return info, fmt.Errorf("keywords: %w", err)
		}
	}
	if info.EDF&EDFGameID != 0 {
		if len(rest) < 8 {
			return info, fmt.Errorf("missing game id")
		}
		info.GameID = binary.LittleEndian.Uint64(rest)
		rest = rest[8:]
	}
	if len(rest) != 0 {
		return info, fmt.Errorf("trailing data: %d bytes", len(rest))
	}
	return info, nil
}
