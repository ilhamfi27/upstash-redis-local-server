package internal

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/gomodule/redigo/redis"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

func (s *Server) handlePublish(ctx *fasthttp.RequestCtx, auth *authResult, segments []string) {
	if len(segments) < 3 {
		s.respond(ctx, errorResult{Error: "ERR wrong number of arguments for publish"}, fasthttp.StatusBadRequest)
		return
	}
	channel := segments[1]
	message := strings.Join(segments[2:], "/")
	res, code := s.executeCommand(auth, "PUBLISH", channel, message)
	s.respond(ctx, res, code)
}

func (s *Server) handleSubscribeSSE(ctx *fasthttp.RequestCtx, auth *authResult, channel string) {
	conn, err := s.streamConn()
	if err != nil {
		s.respond(ctx, errorResult{Error: "ERR redis connection failed"}, fasthttp.StatusServiceUnavailable)
		return
	}
	if auth != nil {
		if err := s.authRedisConn(conn, auth.creds); err != nil {
			conn.Close()
			s.respond(ctx, errorResult{Error: err.Error()}, fasthttp.StatusBadRequest)
			return
		}
	}

	psc := redis.PubSubConn{Conn: conn}
	if err := psc.Subscribe(channel); err != nil {
		conn.Close()
		s.respond(ctx, errorResult{Error: err.Error()}, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetContentType("text/event-stream")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")

	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		// Closing the connection unblocks psc.Receive() when we return.
		defer psc.Close()
		defer conn.Close()

		if _, err := fmt.Fprintf(w, "data: subscribe,%s,1\n\n", channel); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}

		for {
			switch msg := psc.Receive().(type) {
			case redis.Message:
				if _, err := fmt.Fprintf(w, "data: message,%s,%s\n\n", msg.Channel, msg.Data); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					// Client disconnected — stop and release the connection.
					return
				}
			case error:
				s.Logger.Warn("subscribe stream ended", zap.Error(msg))
				return
			}
		}
	})
}

func (s *Server) handleMonitorSSE(ctx *fasthttp.RequestCtx, auth *authResult) {
	conn, err := s.streamConn()
	if err != nil {
		s.respond(ctx, errorResult{Error: "ERR redis connection failed"}, fasthttp.StatusServiceUnavailable)
		return
	}
	if auth != nil {
		if err := s.authRedisConn(conn, auth.creds); err != nil {
			conn.Close()
			s.respond(ctx, errorResult{Error: err.Error()}, fasthttp.StatusBadRequest)
			return
		}
	}

	if err := conn.Send("MONITOR"); err != nil {
		conn.Close()
		s.respond(ctx, errorResult{Error: err.Error()}, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetContentType("text/event-stream")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")

	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		defer conn.Close()

		if _, err := fmt.Fprintf(w, "data: \"OK\"\n\n"); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}

		for {
			line, err := redis.String(conn.Receive())
			if err != nil {
				return
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", line); err != nil {
				return
			}
			if err := w.Flush(); err != nil {
				return
			}
		}
	})
}

func (s *Server) handleHealth(ctx *fasthttp.RequestCtx) {
	conn := s.RedisPool.Get()
	defer conn.Close()
	_, err := conn.Do("PING")
	if err != nil {
		s.writeJSON(ctx, map[string]string{"status": "unhealthy", "redis": err.Error()}, fasthttp.StatusServiceUnavailable)
		return
	}
	s.writeJSON(ctx, map[string]string{"status": "healthy", "redis": "connected"}, fasthttp.StatusOK)
}

func (s *Server) handleReady(ctx *fasthttp.RequestCtx) {
	s.handleHealth(ctx)
}
