#!/usr/bin/env bash
# 单机鉴权冒烟。依赖: curl, jq
# 用法: BASE_URL=http://127.0.0.1:9528 bash scripts/api_security_check.sh
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9528}"
PASS=0
FAIL=0

red() { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }

assert_http() {
  local name="$1" expect="$2" got="$3"
  if [[ "$got" == "$expect" ]]; then
    green "OK  $name (HTTP $got)"
    PASS=$((PASS + 1))
  else
    red "FAIL $name (expect HTTP $expect, got $got)"
    FAIL=$((FAIL + 1))
  fi
}

assert_code0() {
  local name="$1"
  local ec
  ec="$(jq -r '.code' /tmp/qqfarm_body.json)"
  if [[ "$ec" == "0" ]]; then
    green "OK  $name (.code=0)"
    PASS=$((PASS + 1))
  else
    red "FAIL $name (.code=$ec)"
    FAIL=$((FAIL + 1))
  fi
}

req() {
  local method="$1" path="$2"
  shift 2
  curl -sS -o /tmp/qqfarm_body.json -w "%{http_code}" -X "$method" "${BASE_URL}${path}" "$@"
}

echo "==> Unauthenticated farm account list"
CODE="$(req GET /farm/account/list)"
# 项目鉴权失败常返回 HTTP 200 + 业务码 10002
assert_http "no auth farm list http" "200" "$CODE"
ec="$(jq -r '.code' /tmp/qqfarm_body.json)"
if [[ "$ec" == "10002" || "$ec" == "401" ]]; then
  green "OK  no auth farm list (.code=$ec)"
  PASS=$((PASS + 1))
else
  red "FAIL no auth farm list (.code=$ec)"
  FAIL=$((FAIL + 1))
fi

echo "==> Login"
CODE="$(req POST /auth/login -H 'Content-Type: application/json' -d '{"userName":"admin","password":"admin888"}')"
assert_http "login" "200" "$CODE"
assert_code0 "login body"
TOKEN="$(jq -r '.data.token // .data.accessToken // empty' /tmp/qqfarm_body.json)"
if [[ -z "$TOKEN" || "$TOKEN" == "null" ]]; then
  TOKEN="$(jq -r '.data.token // empty' /tmp/qqfarm_body.json)"
fi
if [[ -z "$TOKEN" || "$TOKEN" == "null" ]]; then
  red "FAIL missing token: $(head -c 300 /tmp/qqfarm_body.json)"
  FAIL=$((FAIL + 1))
  echo "Result: PASS=$PASS FAIL=$FAIL"
  exit 1
fi
AUTH=(-H "Authorization: Bearer ${TOKEN}")

echo "==> Auth info"
CODE="$(req GET /auth/info "${AUTH[@]}")"
assert_http "auth info" "200" "$CODE"
assert_code0 "auth info body"

echo "==> Farm account list"
CODE="$(req GET /farm/account/list "${AUTH[@]}")"
assert_http "farm account list" "200" "$CODE"
assert_code0 "farm account list body"

echo "==> Admin list"
CODE="$(req GET /system/admin/list "${AUTH[@]}")"
assert_http "admin list" "200" "$CODE"
assert_code0 "admin list body"

echo "==> Removed routes should 404"
CODE="$(req GET /platform/tenant/list "${AUTH[@]}")"
assert_http "tenant list gone" "404" "$CODE"
CODE="$(req GET /system/attachment/list "${AUTH[@]}")"
assert_http "attachment list gone" "404" "$CODE"
CODE="$(req GET /farm/card/list "${AUTH[@]}")"
assert_http "card list gone" "404" "$CODE"

echo
echo "Result: PASS=$PASS FAIL=$FAIL"
if [[ "$FAIL" -gt 0 ]]; then
  exit 1
fi
