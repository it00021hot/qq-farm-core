#!/usr/bin/env bash
# API 安全与 Swagger 覆盖验收。依赖: curl, jq
# 用法: BASE_URL=http://127.0.0.1:9528 bash scripts/api_security_check.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BASE_URL="${BASE_URL:-http://127.0.0.1:9528}"
SWAGGER_JSON="${SWAGGER_JSON:-$ROOT/docs/swagger.json}"
PASS=0
FAIL=0
TS="$(date +%s)"

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

assert_body_ok() {
  local name="$1" body="$2"
  local code
  code="$(echo "$body" | jq -r '.code // empty')"
  if [[ "$code" == "0" || "$code" == "200" || "$code" == "null" || -z "$code" ]]; then
    # Success code in this project is typically 0 — check message path
    :
  fi
  local ec
  ec="$(echo "$body" | jq -r '.code')"
  # pkg/response Success is usually 0
  if [[ "$ec" == "0" ]]; then
    green "OK  $name (.code=0)"
    PASS=$((PASS + 1))
  else
    red "FAIL $name (.code=$ec body=$(echo "$body" | head -c 200))"
    FAIL=$((FAIL + 1))
  fi
}

assert_body_fail() {
  local name="$1" body="$2"
  local ec
  ec="$(echo "$body" | jq -r '.code')"
  if [[ "$ec" != "0" ]]; then
    green "OK  $name (.code=$ec)"
    PASS=$((PASS + 1))
  else
    red "FAIL $name (expected business error, got success)"
    FAIL=$((FAIL + 1))
  fi
}

req() {
  # req METHOD PATH [extra curl args...]
  local method="$1" path="$2"
  shift 2
  curl -sS -o /tmp/goskel_body.json -w "%{http_code}" -X "$method" "${BASE_URL}${path}" "$@"
}

json_get() {
  jq -r "$1" /tmp/goskel_body.json
}

echo "==> Swagger path coverage"
REQUIRED_PATHS=(
  "/platform/tenant/list"
  "/platform/tenant/detail"
  "/platform/tenant/status"
  "/platform/tenant/bind"
  "/system/admin/list"
  "/system/admin/add"
  "/system/admin/status"
  "/system/platform-user/add"
  "/platform/role/tree"
  "/platform/role/assignable"
  "/platform/role/add"
  "/platform/role/delete"
  "/platform/role/auth"
  "/platform/menu/tree"
  "/platform/menu/add"
  "/platform/menu/delete"
  "/auth/login"
  "/auth/info"
  "/platform/permission/apis"
  "/platform/permission/reload"
)
if [[ ! -f "$SWAGGER_JSON" ]]; then
  red "FAIL swagger.json missing: $SWAGGER_JSON"
  FAIL=$((FAIL + 1))
else
  for p in "${REQUIRED_PATHS[@]}"; do
    if jq -e --arg p "$p" '.paths | has($p)' "$SWAGGER_JSON" >/dev/null; then
      green "OK  swagger has $p"
      PASS=$((PASS + 1))
    else
      red "FAIL swagger missing path $p"
      FAIL=$((FAIL + 1))
    fi
  done
fi

echo "==> Health"
HC="$(curl -sS -o /dev/null -w "%{http_code}" "${BASE_URL}/ping" || true)"
assert_http "GET /ping" "200" "$HC"

echo "==> No token -> 401"
CODE="$(req GET /platform/tenant/list)"
assert_http "GET /platform/tenant/list without token" "401" "$CODE"

echo "==> Super login"
CODE="$(req POST /auth/login -H 'Content-Type: application/json' \
  -d '{"userName":"admin","password":"admin888"}')"
assert_http "super login" "200" "$CODE"
TOKEN_SUPER="$(json_get '.data.token')"
if [[ -z "$TOKEN_SUPER" || "$TOKEN_SUPER" == "null" ]]; then
  red "FATAL: cannot get super token"
  exit 1
fi
AUTH_SUPER=(-H "Authorization: Bearer ${TOKEN_SUPER}" -H "Content-Type: application/json")

echo "==> Super auth/info buttons=*"
CODE="$(req GET /auth/info "${AUTH_SUPER[@]}")"
assert_http "super auth/info" "200" "$CODE"
assert_body_ok "super auth/info body" "$(cat /tmp/goskel_body.json)"
BTNS="$(json_get '.data.buttons | join(",")')"
if [[ "$BTNS" == "*" ]]; then
  green "OK  super buttons=*"
  PASS=$((PASS + 1))
else
  red "FAIL super buttons expect *, got $BTNS"
  FAIL=$((FAIL + 1))
fi

SUFFIX="${TS}"
CODE_A="ta${SUFFIX}"
CODE_B="tb${SUFFIX}"
ACC_A="ta_admin_${SUFFIX}"
ACC_B="tb_admin_${SUFFIX}"
ACC_STAFF="ta_staff_${SUFFIX}"
ACC_OPS="ops_${SUFFIX}"
PASSWD="Passw0rd!"

echo "==> Create tenant A with admin"
CODE="$(req POST /platform/tenant/add "${AUTH_SUPER[@]}" \
  -d "{\"code\":\"${CODE_A}\",\"name\":\"TenantA\",\"max_users\":10,\"admin_account\":\"${ACC_A}\",\"admin_password\":\"${PASSWD}\",\"admin_nick_name\":\"AAdmin\"}")"
assert_http "create tenant A" "200" "$CODE"
assert_body_ok "create tenant A body" "$(cat /tmp/goskel_body.json)"
TID_A="$(json_get '.data.id')"
if [[ -z "$TID_A" || "$TID_A" == "null" || "$CODE" != "200" ]]; then
  red "FATAL: create tenant A failed: $(cat /tmp/goskel_body.json | head -c 400)"
  exit 1
fi

echo "==> Create tenant B with admin"
CODE="$(req POST /platform/tenant/add "${AUTH_SUPER[@]}" \
  -d "{\"code\":\"${CODE_B}\",\"name\":\"TenantB\",\"max_users\":10,\"admin_account\":\"${ACC_B}\",\"admin_password\":\"${PASSWD}\",\"admin_nick_name\":\"BAdmin\"}")"
assert_http "create tenant B" "200" "$CODE"
TID_B="$(json_get '.data.id')"
if [[ -z "$TID_B" || "$TID_B" == "null" || "$CODE" != "200" ]]; then
  red "FATAL: create tenant B failed: $(cat /tmp/goskel_body.json | head -c 400)"
  exit 1
fi

echo "==> Tenant A admin login"
CODE="$(req POST /auth/login -H 'Content-Type: application/json' \
  -d "{\"account\":\"${ACC_A}\",\"password\":\"${PASSWD}\"}")"
assert_http "tenant A login" "200" "$CODE"
TOKEN_A="$(json_get '.data.token')"
if [[ -z "$TOKEN_A" || "$TOKEN_A" == "null" ]]; then
  red "FATAL: cannot get tenant A token: $(cat /tmp/goskel_body.json | head -c 400)"
  exit 1
fi
AUTH_A=(-H "Authorization: Bearer ${TOKEN_A}" -H "Content-Type: application/json")

echo "==> Tenant admin forbidden on platform APIs"
CODE="$(req GET /platform/tenant/list "${AUTH_A[@]}")"
assert_http "tenant admin GET /platform/tenant/list" "403" "$CODE"

CODE="$(req GET /platform/role/tree "${AUTH_A[@]}")"
# business returns 400 with message 仅平台用户 — controller wraps as BadRequest
# Tree service returns error -> BadRequestException 400
if [[ "$CODE" == "400" || "$CODE" == "403" ]]; then
  green "OK  tenant admin GET /platform/role/tree (HTTP $CODE)"
  PASS=$((PASS + 1))
else
  red "FAIL tenant admin GET /platform/role/tree (got $CODE)"
  FAIL=$((FAIL + 1))
fi

CODE="$(req GET /platform/menu/tree "${AUTH_A[@]}")"
if [[ "$CODE" == "400" || "$CODE" == "403" ]]; then
  green "OK  tenant admin GET /platform/menu/tree (HTTP $CODE)"
  PASS=$((PASS + 1))
else
  red "FAIL tenant admin GET /platform/menu/tree (got $CODE)"
  FAIL=$((FAIL + 1))
fi

echo "==> Create staff under tenant A"
CODE="$(req POST /system/admin/add "${AUTH_A[@]}" \
  -d "{\"account\":\"${ACC_STAFF}\",\"password\":\"${PASSWD}\",\"nick_name\":\"Staff\",\"role_ids\":\"4\",\"status\":1}")"
assert_http "create staff" "200" "$CODE"
assert_body_ok "create staff body" "$(cat /tmp/goskel_body.json)"
STAFF_ID="$(json_get '.data.id')"
if [[ -z "$STAFF_ID" || "$STAFF_ID" == "null" || "$CODE" != "200" ]]; then
  # fallback: list by keyword
  CODE="$(req GET "/system/admin/list?keyword=${ACC_STAFF}" "${AUTH_A[@]}")"
  STAFF_ID="$(json_get '.data.list[0].id')"
fi
if [[ -z "$STAFF_ID" || "$STAFF_ID" == "null" ]]; then
  red "FATAL: cannot resolve staff id"
  exit 1
fi

echo "==> Staff login"
CODE="$(req POST /auth/login -H 'Content-Type: application/json' \
  -d "{\"account\":\"${ACC_STAFF}\",\"password\":\"${PASSWD}\"}")"
TOKEN_STAFF="$(json_get '.data.token')"
if [[ -z "$TOKEN_STAFF" || "$TOKEN_STAFF" == "null" ]]; then
  red "FATAL: cannot get staff token: $(cat /tmp/goskel_body.json | head -c 400)"
  exit 1
fi
AUTH_STAFF=(-H "Authorization: Bearer ${TOKEN_STAFF}" -H "Content-Type: application/json")

echo "==> Staff auth/info has admin:list not admin:add"
CODE="$(req GET /auth/info "${AUTH_STAFF[@]}")"
assert_http "staff auth/info" "200" "$CODE"
HAS_LIST="$(jq -r '.data.buttons | index("admin:list") != null' /tmp/goskel_body.json)"
HAS_API="$(jq -r '.data.buttons | index("admin:add") != null' /tmp/goskel_body.json)"
if [[ "$HAS_LIST" == "true" && "$HAS_API" == "false" ]]; then
  green "OK  staff buttons scoped"
  PASS=$((PASS + 1))
else
  red "FAIL staff buttons list=$HAS_LIST api=$HAS_API body=$(head -c 200 /tmp/goskel_body.json)"
  FAIL=$((FAIL + 1))
fi

echo "==> Staff cannot create admin (Casbin)"
CODE="$(req POST /system/admin/add "${AUTH_STAFF[@]}" \
  -d "{\"account\":\"x_${SUFFIX}\",\"password\":\"${PASSWD}\",\"nick_name\":\"X\",\"role_ids\":\"4\",\"status\":1}")"
assert_http "staff POST /system/admin/add" "403" "$CODE"

echo "==> Super grants staff admin:add then staff can POST /system/admin/add"
# role 4 currently has 1,2,4,5,6,16,17 — add resource 3 (admin:add)
CODE="$(req POST /platform/role/auth "${AUTH_SUPER[@]}" \
  -d '{"role_id":4,"resource_ids":"1,2,3,4,5,6,16,17"}')"
assert_http "grant staff admin:add" "200" "$CODE"
assert_body_ok "grant staff body" "$(cat /tmp/goskel_body.json)"
CODE="$(req POST /system/admin/add "${AUTH_STAFF[@]}" \
  -d "{\"account\":\"y_${SUFFIX}\",\"password\":\"${PASSWD}\",\"nick_name\":\"Y\",\"role_ids\":\"4\",\"status\":1}")"
assert_http "staff POST /system/admin/add after grant" "200" "$CODE"
# restore staff seed auth
req POST /platform/role/auth "${AUTH_SUPER[@]}" -d '{"role_id":4,"resource_ids":"1,2,4,5,6,16,17"}' >/dev/null

echo "==> Staff cannot update users (Casbin role write deny)"
CODE="$(req POST /system/admin/modify "${AUTH_STAFF[@]}" \
  -d "{\"id\":${STAFF_ID},\"nick_name\":\"Staff\",\"role_ids\":\"3\",\"status\":1}")"
assert_http "staff PUT assign parent role" "403" "$CODE"

echo "==> Tenant admin cannot assign platform role to staff"
CODE="$(req POST /system/admin/modify "${AUTH_A[@]}" \
  -d "{\"id\":${STAFF_ID},\"nick_name\":\"Staff\",\"role_ids\":\"1\",\"status\":1}")"
assert_http "assign platform role to staff" "400" "$CODE"
assert_body_fail "assign platform role body" "$(cat /tmp/goskel_body.json)"

CODE="$(req POST /system/admin/modify "${AUTH_A[@]}" \
  -d "{\"id\":${STAFF_ID},\"nick_name\":\"Staff\",\"role_ids\":\"2\",\"status\":1}")"
assert_http "assign platform ops role to staff" "400" "$CODE"
assert_body_fail "assign platform ops role body" "$(cat /tmp/goskel_body.json)"

# Tenant admin (role 3) can assign self/child roles 3 and 4
CODE="$(req POST /system/admin/modify "${AUTH_A[@]}" \
  -d "{\"id\":${STAFF_ID},\"nick_name\":\"Staff\",\"role_ids\":\"4\",\"status\":1}")"
assert_http "tenant admin assign staff role" "200" "$CODE"
assert_body_ok "reset staff role body" "$(cat /tmp/goskel_body.json)"

echo "==> Tenant A ignores X-Tenant-ID of B"
CODE="$(req GET /system/admin/list "${AUTH_A[@]}" -H "X-Tenant-ID: ${TID_B}")"
assert_http "tenant A list with X-Tenant-ID=B" "200" "$CODE"
# all users should have tenant_id = TID_A
BAD="$(jq -r --argjson tid "$TID_A" '[(.data.list // [])[] | select(.tenant_id != $tid)] | length' /tmp/goskel_body.json 2>/dev/null || echo error)"
if [[ "$BAD" == "0" ]]; then
  green "OK  tenant A list isolation"
  PASS=$((PASS + 1))
else
  red "FAIL tenant A list leaked other tenants ($BAD)"
  FAIL=$((FAIL + 1))
fi

echo "==> Create platform ops bound only to A"
CODE="$(req POST /system/platform-user/add "${AUTH_SUPER[@]}" \
  -d "{\"account\":\"${ACC_OPS}\",\"password\":\"${PASSWD}\",\"nick_name\":\"Ops\",\"role_ids\":\"2\",\"tenant_ids\":[${TID_A}],\"status\":1}")"
assert_http "create platform ops" "200" "$CODE"
OPS_ID="$(json_get '.data.id')"

CODE="$(req POST /auth/login -H 'Content-Type: application/json' \
  -d "{\"account\":\"${ACC_OPS}\",\"password\":\"${PASSWD}\"}")"
TOKEN_OPS="$(json_get '.data.token')"
AUTH_OPS=(-H "Authorization: Bearer ${TOKEN_OPS}" -H "Content-Type: application/json")

echo "==> Ops without X-Tenant-ID on admin/list -> 400"
CODE="$(req GET /system/admin/list "${AUTH_OPS[@]}")"
assert_http "ops admin/list no tenant header" "400" "$CODE"

echo "==> Ops X-Tenant-ID=B -> 403"
CODE="$(req GET /system/admin/list "${AUTH_OPS[@]}" -H "X-Tenant-ID: ${TID_B}")"
assert_http "ops admin/list tenant B" "403" "$CODE"

echo "==> Ops X-Tenant-ID=A -> 200"
CODE="$(req GET /system/admin/list "${AUTH_OPS[@]}" -H "X-Tenant-ID: ${TID_A}")"
assert_http "ops admin/list tenant A" "200" "$CODE"

echo "==> Super can list A and B"
CODE="$(req GET /system/admin/list "${AUTH_SUPER[@]}" -H "X-Tenant-ID: ${TID_A}")"
assert_http "super list A" "200" "$CODE"
CODE="$(req GET /system/admin/list "${AUTH_SUPER[@]}" -H "X-Tenant-ID: ${TID_B}")"
assert_http "super list B" "200" "$CODE"

echo "==> Disable tenant A then login fails"
CODE="$(req PUT /platform/tenant/list/status "${AUTH_SUPER[@]}" \
  -d "{\"id\":${TID_A},\"status\":2}")"
assert_http "disable tenant A" "200" "$CODE"
CODE="$(req POST /auth/login -H 'Content-Type: application/json' \
  -d "{\"account\":\"${ACC_A}\",\"password\":\"${PASSWD}\"}")"
assert_http "disabled tenant login" "400" "$CODE"
assert_body_fail "disabled tenant login body" "$(cat /tmp/goskel_body.json)"

# re-enable A for quota test
req PUT /platform/tenant/list/status "${AUTH_SUPER[@]}" -d "{\"id\":${TID_A},\"status\":1}" >/dev/null

echo "==> Quota max_users=1 reject new user"
# Tenant A already has admin + staff (>=2). Set max_users=1 should fail update OR set max higher then use new tenant.
CODE_Q="tq${SUFFIX}"
ACC_Q="tq_admin_${SUFFIX}"
CODE="$(req POST /platform/tenant/add "${AUTH_SUPER[@]}" \
  -d "{\"code\":\"${CODE_Q}\",\"name\":\"QuotaT\",\"max_users\":1,\"admin_account\":\"${ACC_Q}\",\"admin_password\":\"${PASSWD}\",\"admin_nick_name\":\"QAdmin\"}")"
assert_http "create quota tenant" "200" "$CODE"
TID_Q="$(json_get '.data.id')"
CODE="$(req POST /auth/login -H 'Content-Type: application/json' \
  -d "{\"account\":\"${ACC_Q}\",\"password\":\"${PASSWD}\"}")"
TOKEN_Q="$(json_get '.data.token')"
AUTH_Q=(-H "Authorization: Bearer ${TOKEN_Q}" -H "Content-Type: application/json")
CODE="$(req POST /system/admin/add "${AUTH_Q[@]}" \
  -d "{\"account\":\"tq_more_${SUFFIX}\",\"password\":\"${PASSWD}\",\"nick_name\":\"More\",\"role_ids\":\"4\",\"status\":1}")"
assert_http "over quota create user" "400" "$CODE"
assert_body_fail "over quota body" "$(cat /tmp/goskel_body.json)"

echo
echo "==== Summary: PASS=$PASS FAIL=$FAIL ===="
if [[ "$FAIL" -gt 0 ]]; then
  exit 1
fi
exit 0
