#!/usr/bin/env sh
set -eu

WEBHOOK_PORT="${E2E_WEBHOOK_PORT:-8888}"
TECHNITIUM_PORT="${E2E_TECHNITIUM_PORT:-5380}"

WEBHOOK_URL="${E2E_WEBHOOK_URL:-http://127.0.0.1:${WEBHOOK_PORT}}"
TECHNITIUM_URL="${E2E_TECHNITIUM_URL:-http://127.0.0.1:${TECHNITIUM_PORT}}"
TECHNITIUM_USER="${E2E_TECHNITIUM_USER:-admin}"
TECHNITIUM_PASSWORD="${E2E_TECHNITIUM_PASSWORD:-admin}"
DNS_SERVER="${E2E_DNS_SERVER:-127.0.0.1}"
DNS_PORT="${E2E_DNS_PORT:-5053}"

ZONE="${E2E_ZONE:-e2e.local}"
RECORD_TYPE="${E2E_RECORD_TYPE:-A}"
RECORD_VALUE="${E2E_RECORD_VALUE:-10.10.10.10}"
RECORD_TTL="${E2E_RECORD_TTL:-60}"
RECORD_NAME="${E2E_RECORD_NAME:-webhook-$(date +%s).${ZONE}}"

CREATE_OUT="$(mktemp)"
DELETE_OUT="$(mktemp)"
trap 'rm -f "${CREATE_OUT}" "${DELETE_OUT}"' EXIT

log() {
  printf '[e2e] %s\n' "$1"
}

fail() {
  printf '[e2e] ERROR: %s\n' "$1" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

wait_http_ok() {
  name="$1"
  url="$2"
  attempts=0
  until curl -fsS "$url" >/dev/null 2>&1; do
    attempts=$((attempts + 1))
    if [ "$attempts" -ge 60 ]; then
      fail "timeout waiting for ${name} at ${url}"
    fi
    sleep 2
  done
}

require_cmd curl
require_cmd sed
require_cmd grep

case "$RECORD_TYPE" in
  A|AAAA|CNAME|TXT) ;;
  *) fail "unsupported E2E_RECORD_TYPE=$RECORD_TYPE (expected A, AAAA, CNAME, or TXT)" ;;
esac

log "waiting for Technitium and webhook health endpoints"
wait_http_ok "Technitium" "${TECHNITIUM_URL}/"
wait_http_ok "webhook" "${WEBHOOK_URL}/healthz"

log "authenticating against Technitium"
LOGIN_JSON="$(curl -fsS -X POST "${TECHNITIUM_URL}/api/user/login" \
  --data-urlencode "user=${TECHNITIUM_USER}" \
  --data-urlencode "pass=${TECHNITIUM_PASSWORD}")"
TOKEN="$(printf '%s' "$LOGIN_JSON" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')"
[ -n "$TOKEN" ] || fail "failed to parse token from Technitium login response: $LOGIN_JSON"

log "ensuring zone exists: ${ZONE}"
ZONE_CREATE_JSON="$(curl -fsS "${TECHNITIUM_URL}/api/zones/create?token=${TOKEN}&zone=${ZONE}&type=Primary" || true)"
printf '%s' "$ZONE_CREATE_JSON" | grep -Eiq '"status":"ok"|already exists' || \
  fail "zone creation failed: ${ZONE_CREATE_JSON}"

if [ "$RECORD_TYPE" = "A" ] || [ "$RECORD_TYPE" = "AAAA" ]; then
  value_key="ipAddress"
elif [ "$RECORD_TYPE" = "CNAME" ]; then
  value_key="cname"
else
  value_key="text"
fi

# Remove any stale test record before create flow.
curl -fsS "${TECHNITIUM_URL}/api/zones/records/delete?token=${TOKEN}&domain=${RECORD_NAME}&zone=${ZONE}&type=${RECORD_TYPE}&${value_key}=${RECORD_VALUE}" >/dev/null 2>&1 || true

CREATE_PAYLOAD=$(cat <<EOF
{"create":[{"dnsName":"${RECORD_NAME}","targets":["${RECORD_VALUE}"],"recordType":"${RECORD_TYPE}","recordTTL":${RECORD_TTL}}],"updateOld":[],"updateNew":[],"delete":[]}
EOF
)

log "creating record via webhook: ${RECORD_NAME} ${RECORD_TYPE} ${RECORD_VALUE}"
CREATE_STATUS="$(curl -sS -o "$CREATE_OUT" -w '%{http_code}' \
  -X POST "${WEBHOOK_URL}/records" \
  -H 'Content-Type: application/json' \
  --data "$CREATE_PAYLOAD")"
[ "$CREATE_STATUS" = "204" ] || fail "webhook create returned HTTP ${CREATE_STATUS}: $(cat "$CREATE_OUT")"

sleep 2

log "verifying record appears in Technitium API"
ZONE_RECORDS="$(curl -fsS "${TECHNITIUM_URL}/api/zones/records/get?token=${TOKEN}&domain=${ZONE}&listZone=true")"
printf '%s' "$ZONE_RECORDS" | grep -Fq "$RECORD_NAME" || fail "record name not found in Technitium records response"
printf '%s' "$ZONE_RECORDS" | grep -Fq "$RECORD_VALUE" || fail "record value not found in Technitium records response"

if command -v dig >/dev/null 2>&1; then
  log "verifying DNS answer with dig"
  DNS_ANSWER="$(dig +short @"${DNS_SERVER}" -p "${DNS_PORT}" "${RECORD_NAME}" "${RECORD_TYPE}" | tr -d '\r')"
  printf '%s' "$DNS_ANSWER" | grep -Fq "$RECORD_VALUE" || fail "dig did not return expected value for ${RECORD_NAME}"
fi

DELETE_PAYLOAD=$(cat <<EOF
{"create":[],"updateOld":[],"updateNew":[],"delete":[{"dnsName":"${RECORD_NAME}","targets":["${RECORD_VALUE}"],"recordType":"${RECORD_TYPE}","recordTTL":${RECORD_TTL}}]}
EOF
)

log "deleting record via webhook"
DELETE_STATUS="$(curl -sS -o "$DELETE_OUT" -w '%{http_code}' \
  -X POST "${WEBHOOK_URL}/records" \
  -H 'Content-Type: application/json' \
  --data "$DELETE_PAYLOAD")"
[ "$DELETE_STATUS" = "204" ] || fail "webhook delete returned HTTP ${DELETE_STATUS}: $(cat "$DELETE_OUT")"

sleep 2

log "verifying record deletion in Technitium API"
ZONE_RECORDS_AFTER_DELETE="$(curl -fsS "${TECHNITIUM_URL}/api/zones/records/get?token=${TOKEN}&domain=${ZONE}&listZone=true")"
if printf '%s' "$ZONE_RECORDS_AFTER_DELETE" | grep -Fq "$RECORD_NAME"; then
  fail "record still present after delete"
fi

log "e2e scenario passed"
