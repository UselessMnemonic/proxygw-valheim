package frontends

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"

	"github.com/UselessMnemonic/proxygw-valheim/a2s"
	"github.com/UselessMnemonic/proxygw/pkg/config"
	"github.com/UselessMnemonic/proxygw/pkg/frontend"
)

const (
	a2sValheimAppID     = 0
	a2sValheimGameID    = 892970
	a2sDefaultChallenge = 0x70786777 // pxgw
)

var a2sDefaultSteamID = a2s.SteamID(a2s.UniversePublic, a2s.AccountTypeAnonGameServer, 0, 0)

type A2SHandler struct {
	address netip.AddrPort
	info    a2s.Info
	logger  *slog.Logger
	warm    chan struct{}

	conn   *net.UDPConn
	done   chan error
	closed bool
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
	buf := make([]byte, a2s.ReadBufferSize)
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

func (h *A2SHandler) response(packet []byte) ([]byte, a2s.Query, bool) {
	if len(packet) < len(a2s.Header)+1 || !bytes.HasPrefix(packet, []byte(a2s.Header)) {
		return nil, "", false
	}

	switch packet[len(a2s.Header)] {
	case 'T':
		if !bytes.HasPrefix(packet, []byte(a2s.InfoRequest)) {
			h.logger.Debug("a2s info query ignored", "reason", "bad_payload", "bytes", len(packet))
			return nil, "", false
		}
		if !a2s.InfoHasChallenge(packet, a2sDefaultChallenge) {
			h.logger.Debug("a2s info query challenged", "bytes", len(packet))
			return a2s.ChallengeResponse(a2sDefaultChallenge), a2s.QueryInfo, true
		}
		h.logger.Debug("a2s info query received", "bytes", len(packet))
		response, _ := h.info.MarshalBinary()
		return response, a2s.QueryInfo, true
	case 'U':
		if !a2s.HasChallenge(packet, a2sDefaultChallenge) {
			h.logger.Debug("a2s player query challenged", "bytes", len(packet))
			return a2s.ChallengeResponse(a2sDefaultChallenge), a2s.QueryPlayer, true
		}
		h.logger.Debug("a2s player query received", "bytes", len(packet))
		return a2s.PlayerResponse(), a2s.QueryPlayer, true
	case 'V':
		if !a2s.HasChallenge(packet, a2sDefaultChallenge) {
			h.logger.Debug("a2s rules query challenged", "bytes", len(packet))
			return a2s.ChallengeResponse(a2sDefaultChallenge), a2s.QueryRules, true
		}
		h.logger.Debug("a2s rules query received", "bytes", len(packet))
		return a2s.RulesResponse(), a2s.QueryRules, true
	case 'W':
		h.logger.Debug("a2s challenge query received", "bytes", len(packet))
		return a2s.ChallengeResponse(a2sDefaultChallenge), a2s.QueryChallenge, true
	case 'i':
		h.logger.Debug("a2a ping query received", "bytes", len(packet))
		return a2s.PingResponse(), a2s.QueryPing, true
	default:
		h.logger.Debug("a2s packet ignored", "reason", "unknown_query", "query_byte", packet[len(a2s.Header)], "bytes", len(packet))
		return nil, "", false
	}
}

func defaultGamePort(address netip.AddrPort) uint16 {
	if address.Port() == 0 {
		return 0
	}
	return address.Port() - 1
}

func newA2SInfo(address netip.AddrPort, options map[string]any) (a2s.Info, error) {
	name, err := requiredStringOption(options, "name")
	if err != nil {
		return a2s.Info{}, err
	}
	gameVersion, err := requiredStringOption(options, "version")
	if err != nil {
		return a2s.Info{}, err
	}

	info := a2s.Info{
		Name:        name,
		Map:         name,
		Folder:      "valheim",
		Game:        "",
		AppID:       a2sValheimAppID,
		MaxPlayers:  10,
		ServerType:  a2s.ServerTypeDedicated,
		Environment: a2s.EnvironmentLinux,
		Visibility:  a2s.VisibilityPrivate,
		Version:     a2s.InfoVersion,
		EDF:         a2s.EDFPort | a2s.EDFSteamID | a2s.EDFKeywords | a2s.EDFGameID,
		Port:        defaultGamePort(address),
		SteamID:     a2sDefaultSteamID,
		GameID:      a2sValheimGameID,
	}
	info.Keywords = defaultKeywords(gameVersion)

	password, err := boolOption(options, "password", true)
	if err != nil {
		return info, err
	}
	info.Visibility = a2s.VisibilityPublic
	if password {
		info.Visibility = a2s.VisibilityPrivate
	}

	vac, err := boolOption(options, "vac", false)
	if err != nil {
		return info, err
	}
	info.VAC = a2s.VACUnsecured
	if vac {
		info.VAC = a2s.VACSecured
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

func defaultKeywords(gameVersion string) string {
	return fmt.Sprintf("g=%s,n=36,m=", gameVersion)
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

// NewA2SHandler creates a Valheim Steam query frontend. It answers A2S probes
// without warming the target, which keeps Steam/master-server polling separate
// from player connection attempts.
func NewA2SHandler(name string, protocol config.Protocol, address netip.AddrPort, options map[string]any) (frontend.Handler, error) {
	if protocol != config.ProtocolUDP {
		return nil, fmt.Errorf("valheim status frontend requires udp protocol")
	}

	info, err := newA2SInfo(address, options)
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
