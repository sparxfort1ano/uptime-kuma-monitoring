#!/bin/sh
set -eu

AUTH="${GRAFANA_USER}:${GRAFANA_PASSWORD}"
URL="http://grafana:3000"

if ! curl -fsS -u "$AUTH" "$URL/api/datasources/name/Prometheus" >/dev/null 2>&1; then
    curl -fsS -u "$AUTH" -X POST "$URL/api/datasources" \
    -H "Content-Type: application/json" \
    -d '{"name":"Prometheus","type":"prometheus","url":"http://prometheus:9090","access":"proxy","isDefault":true}'
fi

DS_UID=$(curl -fsS -u "$AUTH" "$URL/api/datasources/name/Prometheus" | jq -r .uid)

for file in /grafana/dashboards/*.json; do
  echo "Uploading $file..."
  jq --arg uid "$DS_UID" '
    walk(if type == "string"
         then gsub("\\$\\{DS_[A-Za-z0-9_-]+\\}"; $uid)
         else . end)
    | del(.id, .__inputs)
    | {dashboard: ., overwrite: true}
  ' "$file" | \
  curl -fsS -u "$AUTH" -X POST "$URL/api/dashboards/db" \
    -H "Content-Type: application/json" -d @-
  echo
done