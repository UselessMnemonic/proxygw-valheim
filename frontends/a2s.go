package frontends

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"runtime"

	"github.com/UselessMnemonic/proxygw/pkg/config"
	"github.com/UselessMnemonic/proxygw/pkg/frontend"
)

const (
	a2sReadBufferSize = 1400
	a2sHeader         = "\xff\xff\xff\xff"
	a2sInfoRequest    = "\xff\xff\xff\xffTSource Engine Query\x00"
	a2sDefaultAppID   = 41002 // Steam AppID 892970 truncated to the A2S uint16 field.
	a2sChallenge      = 0x70726f78
)

type a2sQuery string

const (
	a2sQueryInfo      a2sQuery = "a2s_info"
	a2sQueryPlayer    a2sQuery = "a2s_player"
	a2sQueryRules     a2sQuery = "a2s_rules"
	a2sQueryChallenge a2sQuery = "a2s_challenge"
	a2sQueryPing      a2sQuery = "a2a_ping"
)

type A2SHandler struct {
	address netip.AddrPort
	info    a2sInfo
	logger  *slog.Logger
	warm    chan struct{}

	conn   *net.UDPConn
	done   chan error
	closed bool
}

type a2sInfo struct {
	Name       string
	Map        string
	Folder     string
	Game       string
	AppID      uint16
	Players    uint8
	MaxPlayers uint8
	Bots       uint8
	Password   bool
	VAC        bool
	Version    string
}

func (h *A2SHandler) Start() error {
	if h.closed {
		return fmt.Errorf("valheim status frontend is closed")
	}
	if h.conn != nil {
		h.logger.Info("valheim status frontend already started", "listen", h.address.String())
		return nil
	}

	h.logger.Info("starting valheim status frontend", "listen", h.address.String())
	conn, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(h.address))
	if err != nil {
		h.logger.Error("valheim status frontend listen failed", "listen", h.address.String(), "err", err)
		return err
	}

	h.conn = conn
	h.done = make(chan error, 1)
	go func() {
		h.done <- h.serve(conn)
	}()

	h.logger.Info("valheim status frontend started", "listen", h.address.String())
	return nil
}

func (h *A2SHandler) Stop() error {
	if h.conn == nil {
		h.logger.Info("valheim status frontend already stopped")
		return nil
	}

	h.logger.Info("stopping valheim status frontend")
	err := h.conn.Close()
	h.conn = nil

	serveErr := <-h.done
	h.done = nil
	err = errors.Join(err, serveErr)
	if err != nil {
		h.logger.Info("valheim status frontend stopped with error", "err", err)
		return err
	}

	h.logger.Info("valheim status frontend stopped")
	return nil
}

func (h *A2SHandler) Close() error {
	if h.closed {
		return nil
	}
	h.closed = true
	return h.Stop()
}

// ShouldWarm is intentionally inert: Steam server-list/A2S polling must not
// wake a sleeping Valheim server.
func (h *A2SHandler) ShouldWarm() <-chan struct{} {
	return h.warm
}

func (h *A2SHandler) serve(conn *net.UDPConn) error {
	buf := make([]byte, a2sReadBufferSize)
	for {
		n, addr, err := conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}

		response, query, ok := h.response(buf[:n])
		if !ok {
			h.logger.Debug("non-a2s packet ignored", "remote", addr.String(), "bytes", n)
			continue
		}

		if _, err := conn.WriteToUDPAddrPort(response, addr); err != nil {
			h.logger.Debug("a2s response failed", "remote", addr.String(), "query", string(query), "err", err)
			continue
		}
		h.logger.Debug("a2s response sent", "remote", addr.String(), "query", string(query), "bytes_in", n, "bytes_out", len(response))
	}
}

func (h *A2SHandler) response(packet []byte) ([]byte, a2sQuery, bool) {
	if len(packet) < len(a2sHeader)+1 || !bytes.HasPrefix(packet, []byte(a2sHeader)) {
		return nil, "", false
	}

	switch packet[len(a2sHeader)] {
	case 'T':
		if !bytes.HasPrefix(packet, []byte(a2sInfoRequest)) {
			h.logger.Debug("a2s info query ignored", "reason", "bad_payload", "bytes", len(packet))
			return nil, "", false
		}
		h.logger.Debug("a2s info query received", "bytes", len(packet))
		return h.info.response(), a2sQueryInfo, true
	case 'U':
		h.logger.Debug("a2s player query received", "bytes", len(packet))
		return a2sChallengeResponse(), a2sQueryPlayer, true
	case 'V':
		h.logger.Debug("a2s rules query received", "bytes", len(packet))
		return a2sChallengeResponse(), a2sQueryRules, true
	case 'W':
		h.logger.Debug("a2s challenge query received", "bytes", len(packet))
		return a2sChallengeResponse(), a2sQueryChallenge, true
	case 'i':
		h.logger.Debug("a2a ping query received", "bytes", len(packet))
		return []byte(a2sHeader + "j00000000000000\x00"), a2sQueryPing, true
	default:
		h.logger.Debug("a2s packet ignored", "reason", "unknown_query", "query_byte", packet[len(a2sHeader)], "bytes", len(packet))
		return nil, "", false
	}
}

func (i a2sInfo) response() []byte {
	var b bytes.Buffer
	b.WriteString(a2sHeader)
	b.WriteByte('I')
	b.WriteByte(17)
	writeA2SString(&b, i.Name)
	writeA2SString(&b, i.Map)
	writeA2SString(&b, i.Folder)
	writeA2SString(&b, i.Game)
	_ = binary.Write(&b, binary.LittleEndian, i.AppID)
	b.WriteByte(i.Players)
	b.WriteByte(i.MaxPlayers)
	b.WriteByte(i.Bots)
	b.WriteByte('d')
	b.WriteByte(a2sEnvironment())
	writeA2SBool(&b, i.Password)
	writeA2SBool(&b, i.VAC)
	writeA2SString(&b, i.Version)
	b.WriteByte(0)
	return b.Bytes()
}

func a2sChallengeResponse() []byte {
	var b bytes.Buffer
	b.WriteString(a2sHeader)
	b.WriteByte('A')
	_ = binary.Write(&b, binary.LittleEndian, int32(a2sChallenge))
	return b.Bytes()
}

func writeA2SString(b *bytes.Buffer, value string) {
	b.WriteString(value)
	b.WriteByte(0)
}

func writeA2SBool(b *bytes.Buffer, value bool) {
	if value {
		b.WriteByte(1)
		return
	}
	b.WriteByte(0)
}

func a2sEnvironment() byte {
	if runtime.GOOS == "windows" {
		return 'w'
	}
	if runtime.GOOS == "darwin" {
		return 'm'
	}
	return 'l'
}

func newA2SInfo(options map[string]any) (a2sInfo, error) {
	name, err := requiredStringOption(options, "name")
	if err != nil {
		return a2sInfo{}, err
	}
	mapName, err := requiredStringOption(options, "map")
	if err != nil {
		return a2sInfo{}, err
	}

	info := a2sInfo{
		Name:       name,
		Map:        mapName,
		Folder:     stringOption(options, "folder", "valheim"),
		Game:       stringOption(options, "game", "Valheim"),
		AppID:      a2sDefaultAppID,
		MaxPlayers: 10,
		Version:    stringOption(options, "version", ""),
	}

	if info.AppID, err = uint16Option(options, "app_id", info.AppID); err != nil {
		return info, err
	}
	if info.Players, err = uint8Option(options, "players", info.Players); err != nil {
		return info, err
	}
	if info.MaxPlayers, err = uint8Option(options, "max_players", info.MaxPlayers); err != nil {
		return info, err
	}
	if info.Bots, err = uint8Option(options, "bots", info.Bots); err != nil {
		return info, err
	}
	if info.Password, err = boolOption(options, "password", false); err != nil {
		return info, err
	}
	if info.VAC, err = boolOption(options, "vac", false); err != nil {
		return info, err
	}
	return info, nil
}

func requiredStringOption(options map[string]any, key string) (string, error) {
	value, ok := options[key].(string)
	if !ok || value == "" {
		return "", fmt.Errorf("valheim status frontend option %s is required", key)
	}
	return value, nil
}

func stringOption(options map[string]any, key string, fallback string) string {
	value, ok := options[key].(string)
	if !ok {
		return fallback
	}
	return value
}

func boolOption(options map[string]any, key string, fallback bool) (bool, error) {
	value, ok := options[key]
	if !ok {
		return fallback, nil
	}
	typed, ok := value.(bool)
	if !ok {
		return fallback, fmt.Errorf("valheim status frontend option %s must be a boolean", key)
	}
	return typed, nil
}

func uint8Option(options map[string]any, key string, fallback uint8) (uint8, error) {
	value, ok := options[key]
	if !ok {
		return fallback, nil
	}

	n, ok := intOption(value)
	if !ok || n < 0 || n > 255 {
		return fallback, fmt.Errorf("valheim status frontend option %s must be an integer from 0 to 255", key)
	}
	return uint8(n), nil
}

func uint16Option(options map[string]any, key string, fallback uint16) (uint16, error) {
	value, ok := options[key]
	if !ok {
		return fallback, nil
	}

	n, ok := intOption(value)
	if !ok || n < 0 || n > 65535 {
		return fallback, fmt.Errorf("valheim status frontend option %s must be an integer from 0 to 65535", key)
	}
	return uint16(n), nil
}

func intOption(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int8:
		return int64(typed), true
	case int16:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case uint:
		if uint64(typed) > 1<<63-1 {
			return 0, false
		}
		return int64(typed), true
	case uint8:
		return int64(typed), true
	case uint16:
		return int64(typed), true
	case uint32:
		return int64(typed), true
	case uint64:
		if typed > 1<<63-1 {
			return 0, false
		}
		return int64(typed), true
	default:
		return 0, false
	}
}

// NewA2SHandler creates a Valheim Steam query frontend. It answers A2S probes
// without warming the target, which keeps Steam/master-server polling separate
// from player connection attempts.
func NewA2SHandler(name string, protocol config.Protocol, address netip.AddrPort, options map[string]any) (frontend.Handler, error) {
	if protocol != config.ProtocolUDP {
		return nil, fmt.Errorf("valheim status frontend requires udp protocol")
	}

	info, err := newA2SInfo(options)
	if err != nil {
		return nil, err
	}

	return &A2SHandler{
		address: address,
		info:    info,
		logger:  slog.Default().With("handler", "status", "frontend", name),
		warm:    make(chan struct{}),
	}, nil
}
