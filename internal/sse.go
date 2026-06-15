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
	conn := s.RedisPool.Get()
	defer conn.Close()
	if auth != nil {
		if err := s.authRedisConn(conn, auth.creds); err != nil {
			s.respond(ctx, errorResult{Error: err.Error()}, fasthttp.StatusBadRequest)
			return
		}
	}

	psc := redis.PubSubConn{Conn: conn}
	if err := psc.Subscribe(channel); err != nil {
		s.respond(ctx, errorResult{Error: err.Error()}, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetContentType("text/event-stream")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")

	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		fmt.Fprintf(w, "data: subscribe,%s,1\n\n", channel)
		w.Flush()

		for {
			switch msg := psc.Receive().(type) {
			case redis.Message:
				fmt.Fprintf(w, "data: message,%s,%s\n\n", msg.Channel, msg.Data)
				w.Flush()
			case error:
				s.Logger.Warn("subscribe stream ended", zap.Error(msg))
				return
			}
		}
	})
}

func (s *Server) handleMonitorSSE(ctx *fasthttp.RequestCtx, auth *authResult) {
	conn := s.RedisPool.Get()
	defer conn.Close()
	if auth != nil {
		if err := s.authRedisConn(conn, auth.creds); err != nil {
			s.respond(ctx, errorResult{Error: err.Error()}, fasthttp.StatusBadRequest)
			return
		}
	}

	if err := conn.Send("MONITOR"); err != nil {
		s.respond(ctx, errorResult{Error: err.Error()}, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetContentType("text/event-stream")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")

	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		fmt.Fprintf(w, "data: \"OK\"\n\n")
		w.Flush()

		for {
			line, err := redis.String(conn.Receive())
			if err != nil {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", line)
			w.Flush()
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
