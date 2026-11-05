#!/bin/sh
set -eu

TARGET=uptime-kuma
QUERY='up{job="uptime-kuma"}'
PROM="${PROM_URL:-http://localhost:9090}"
TIMEOUT="${TIMEOUT:-90}"
LOOK="${LOOK_SECONDS:-15}"

metric() {
    curl -fsS -G "$PROM/api/v1/query" --data-urlencode "query=$QUERY" \
    | jq -r '.data.result[0].value[1] // "none"'
}

wait_for() {   # wait_for <ожидаемое значение>
    waited=0
    while [ "$waited" -lt "$TIMEOUT" ]; do
    [ "$(metric)" = "$1" ] && return 0
    sleep 3; waited=$((waited + 3))
    done
    return 1
}

fail() { echo "FAIL: $*"; exit 1; }
trap 'docker start "$TARGET" >/dev/null 2>&1 || true' EXIT

echo "=== [0/4] Исходное состояние: должно быть UP ==="
wait_for 1 || fail "$TARGET не в UP до начала теста"

echo "=== [1/4] docker stop $TARGET ==="
docker stop "$TARGET" >/dev/null
wait_for 0 || fail "Prometheus не зафиксировал падение за ${TIMEOUT}s"
echo "OK: падение зафиксировано. Смотрите Grafana, пауза ${LOOK}s"
sleep "$LOOK"

echo "=== [2/4] docker start $TARGET ==="
docker start "$TARGET" >/dev/null
wait_for 1 || fail "$TARGET не вернулся в UP за ${TIMEOUT}s"

echo "=== [3/4] убиваем процессы внутри контейнера (имитация краша) ==="
sleep 15   # restart policy работает, только если контейнер проработал >10s
BEFORE=$(docker inspect -f '{{.RestartCount}}' "$TARGET")
docker exec "$TARGET" sh -c '
    for p in /proc/[0-9]*; do
        pid=${p#/proc/}
        [ "$pid" != "1" ] && [ "$pid" != "$$" ] && kill -9 "$pid" 2>/dev/null
    done' || true

echo "=== [4/4] Ждём автоподъёма ==="
waited=0
while [ "$(docker inspect -f '{{.RestartCount}}' "$TARGET")" -le "$BEFORE" ]; do
    waited=$((waited + 2))
    [ "$waited" -ge "$TIMEOUT" ] && fail "restart-policy не сработала за ${TIMEOUT}s"
    sleep 2
done
echo "OK: Docker перезапустил контейнер"
wait_for 1 || fail "метрика не вернулась в UP"

echo "=== PASS ==="