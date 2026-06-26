#!/bin/sh
# End-to-end smoke test for the new features.
# Boots a real Redis + the server and exercises each feature over HTTP.
set -e

echo "==> Installing redis + curl"
apk add --no-cache redis curl >/dev/null 2>&1

echo "==> Starting Redis"
redis-server --daemonize yes --save "" >/dev/null
sleep 1

echo "==> Building server"
cd /app
go build -o /tmp/upstash . 

TOKEN=local-dev-token
AUTH="Authorization: Bearer $TOKEN"

pass() { echo "  PASS: $1"; }
fail() { echo "  FAIL: $1"; FAILED=1; }

# ------------------------------------------------------------------
# Instance A — QStash emulator + command recording
# ------------------------------------------------------------------
/tmp/upstash --token $TOKEN --addr :8000 --redis 127.0.0.1:6379 \
  --enable-qstash --record /tmp/rec.jsonl >/tmp/a.log 2>&1 &
A_PID=$!
sleep 2

echo ""
echo "================ QStash emulator (:8000) ================"
# The destination is the server's own SET endpoint; we forward the auth header
# so the delivered POST is authorised. Empty body keeps the path command clean.
DEST="http://127.0.0.1:8000/SET/qstash_demo/delivered_ok"
PUB=$(curl -s -X POST "http://127.0.0.1:8000/v2/publish/?url=$DEST" \
  -H "$AUTH" -H "Upstash-Forward-Authorization: Bearer $TOKEN" --data-raw '')
echo "publish response : $PUB"
echo "$PUB" | grep -q '"messageId"' && pass "publish returned a messageId" || fail "publish"

echo "waiting for background delivery..."
sleep 3

GOT=$(curl -s "http://127.0.0.1:8000/GET/qstash_demo" -H "$AUTH")
echo "delivered key    : $GOT"
echo "$GOT" | grep -q "delivered_ok" && pass "message was delivered to destination" || fail "delivery"

MSGS=$(curl -s "http://127.0.0.1:8000/v2/messages" -H "$AUTH")
echo "messages list    : $MSGS"
echo "$MSGS" | grep -q '"state":"delivered"' && pass "message marked delivered" || fail "message state"

echo ""
echo "================ Command recording ================"
echo "recorded file /tmp/rec.jsonl:"
sed 's/^/    /' /tmp/rec.jsonl
grep -q '"SET"' /tmp/rec.jsonl 2>/dev/null && pass "commands were recorded" || \
  ( grep -q 'SET' /tmp/rec.jsonl && pass "commands were recorded" ) || fail "recording"

kill $A_PID 2>/dev/null || true

# ------------------------------------------------------------------
# Instance B — strict Upstash parity mode
# ------------------------------------------------------------------
/tmp/upstash --token $TOKEN --addr :8001 --redis 127.0.0.1:6379 \
  --strict-upstash >/tmp/b.log 2>&1 &
B_PID=$!
sleep 2

echo ""
echo "================ Strict Upstash mode (:8001) ================"
SUB=$(curl -s "http://127.0.0.1:8001/SUBSCRIBE/news" -H "$AUTH")
echo "SUBSCRIBE        : $SUB"
echo "$SUB" | grep -q "strict mode" && pass "unsupported command blocked" || fail "strict block"

PINGOK=$(curl -s "http://127.0.0.1:8001/PING" -H "$AUTH")
echo "PING             : $PINGOK"
echo "$PINGOK" | grep -q "PONG" && pass "supported command still works" || fail "strict allow"

kill $B_PID 2>/dev/null || true

# ------------------------------------------------------------------
# Instance C — chaos injection (latency + forced errors)
# ------------------------------------------------------------------
/tmp/upstash --token $TOKEN --addr :8002 --redis 127.0.0.1:6379 \
  --inject-latency 300 --inject-error-rate 1.0 >/tmp/c.log 2>&1 &
C_PID=$!
sleep 2

echo ""
echo "================ Chaos injection (:8002) ================"
RESP=$(curl -s -w "|status=%{http_code}|time=%{time_total}s" "http://127.0.0.1:8002/PING" -H "$AUTH")
echo "PING (latency+err): $RESP"
echo "$RESP" | grep -q "status=500" && pass "error injection returned 500" || fail "error injection"
echo "$RESP" | grep -q "chaos injection" && pass "error body explains chaos" || fail "error body"
# time_total should be >= 0.3s due to injected latency
T=$(echo "$RESP" | sed 's/.*time=//;s/s.*//')
awk "BEGIN{exit !($T >= 0.3)}" && pass "latency injection added >=300ms" || fail "latency injection"

kill $C_PID 2>/dev/null || true

echo ""
if [ "$FAILED" = "1" ]; then
  echo "RESULT: SOME CHECKS FAILED"
  exit 1
fi
echo "RESULT: ALL FEATURE CHECKS PASSED"
