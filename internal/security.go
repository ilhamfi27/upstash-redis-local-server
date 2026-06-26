package internal

import (
	"fmt"
	"net"
	"strings"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

// SecurityConfig controls optional hardening for local development.
type SecurityConfig struct {
	DisableQueryToken      bool
	CORSOrigin             string
	RequireDashboardAuth   bool
	BlockDangerousCommands bool
	StrictUpstash          bool
}

// strictBlocked are commands Upstash Cloud rejects over its REST API. Blocking
// them locally catches "works on my machine, breaks in prod" bugs early.
var strictBlocked = map[string]bool{
	"SUBSCRIBE": true, "UNSUBSCRIBE": true, "PSUBSCRIBE": true, "PUNSUBSCRIBE": true,
	"BLPOP": true, "BRPOP": true, "BLMOVE": true, "BLMPOP": true, "BRPOPLPUSH": true,
	"BZPOPMIN": true, "BZPOPMAX": true, "BZMPOP": true, "WAIT": true,
	"MONITOR": true, "SYNC": true, "PSYNC": true, "SLAVEOF": true, "REPLICAOF": true,
	"SELECT": true, "SWAPDB": true, "MOVE": true, "DEBUG": true, "SHUTDOWN": true,
	"FAILOVER": true, "CLIENT": true, "RESET": true, "MULTI": true, "EXEC": true,
	"DISCARD": true, "WATCH": true, "UNWATCH": true, "SUBSCRIBE_SHARD": true,
}

var weakTokens = map[string]bool{
	"upstash":            true,
	"local-dev-token":    true,
	"local-readonly-token": true,
	"test-token":         true,
	"test":               true,
	"dev":                true,
	"password":           true,
}

var dangerousCommands = map[string]bool{
	"FLUSHALL": true, "FLUSHDB": true, "CONFIG": true, "SHUTDOWN": true,
	"DEBUG": true, "SLAVEOF": true, "REPLICAOF": true, "MIGRATE": true,
}

// ResolveListenAddr applies localhost-only binding while preserving the port.
func ResolveListenAddr(addr string, localhostOnly bool) string {
	if !localhostOnly {
		return addr
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// bare ":8000" or "8000"
		if strings.HasPrefix(addr, ":") {
			return "127.0.0.1" + addr
		}
		return "127.0.0.1:" + addr
	}
	if host == "" || host == "0.0.0.0" {
		return net.JoinHostPort("127.0.0.1", port)
	}
	return addr
}

func isWeakToken(token string) bool {
	if token == "" {
		return false
	}
	return weakTokens[strings.ToLower(token)]
}

func (s *Server) isDangerousCommandBlocked(commandName string) error {
	cmd := strings.ToUpper(commandName)
	if s.Security.BlockDangerousCommands && dangerousCommands[cmd] {
		return fmt.Errorf("ERR command '%s' is blocked by local security policy", cmd)
	}
	if s.Security.StrictUpstash && strictBlocked[cmd] {
		return fmt.Errorf("ERR command '%s' is not supported by Upstash REST (strict mode)", cmd)
	}
	return nil
}

// LogSecurityWarnings prints startup guidance for insecure configurations.
func LogSecurityWarnings(logger *zap.Logger, addr, apiToken, readOnlyToken string, sec SecurityConfig, secureMode bool) {
	if isWeakToken(apiToken) {
		logger.Warn("using a well-known API token — generate a random token for shared networks",
			zap.String("hint", "openssl rand -base64 32"),
		)
	}
	if readOnlyToken != "" && isWeakToken(readOnlyToken) {
		logger.Warn("using a well-known read-only token — replace with a random value")
	}

	host, _, err := net.SplitHostPort(addr)
	if err == nil && (host == "" || host == "0.0.0.0") {
		logger.Warn("listening on all network interfaces — use --localhost-only or bind 127.0.0.1 to restrict access")
	}

	if sec.DisableQueryToken {
		logger.Info("query token auth disabled — use Authorization: Bearer header only")
	}
	if sec.BlockDangerousCommands {
		logger.Info("dangerous Redis commands blocked (FLUSHALL, CONFIG, SHUTDOWN, etc.)")
	}
	if sec.RequireDashboardAuth {
		logger.Info("dashboard API requires authentication")
	}
	if sec.StrictUpstash {
		logger.Info("strict Upstash parity mode — rejecting commands unsupported by Upstash REST")
	}
	if sec.CORSOrigin != "" && sec.CORSOrigin != "*" {
		logger.Info("CORS restricted", zap.String("origin", sec.CORSOrigin))
	}
	if secureMode {
		logger.Info("secure mode enabled")
	}
}

func (s *Server) corsOrigin() string {
	if s.Security.CORSOrigin == "" {
		return "*"
	}
	return s.Security.CORSOrigin
}

func (s *Server) requireDashboardAuth() bool {
	return s.Security.RequireDashboardAuth
}

func (s *Server) disableQueryToken() bool {
	return s.Security.DisableQueryToken
}

func (s *Server) authenticateDashboard(ctx *fasthttp.RequestCtx) error {
	if !s.requireDashboardAuth() {
		return nil
	}
	_, err := s.authenticate(ctx)
	return err
}
