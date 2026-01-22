#!/bin/bash

# Debug version - captures full responses for analysis
# This will help identify why tests are failing

BASE_URL="http://localhost:8019/api/v1/backoffice"
SUPER_ADMIN_EMAIL="superadmin@example.com"
SUPER_ADMIN_PASSWORD="superadmin12345"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_section() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}\n"
}

print_info() {
    echo -e "${YELLOW}ℹ INFO:${NC} $1"
}

# Helper to make requests and handle both JSON and non-JSON responses
make_request() {
    local METHOD=$1
    local URL=$2
    local AUTH=$3
    local DATA=$4
    
    if [ -z "$DATA" ]; then
        RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X $METHOD "$URL" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $AUTH")
    else
        RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X $METHOD "$URL" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $AUTH" \
          -d "$DATA")
    fi
    
    HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)
    BODY=$(echo "$RESPONSE" | sed '/HTTP_CODE:/d')
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Request: $METHOD $URL"
    if [ ! -z "$DATA" ]; then
        echo "Body: $DATA"
    fi
    echo "HTTP Status: $HTTP_CODE"
    echo "Response:"
    
    if echo "$BODY" | jq empty 2>/dev/null; then
        echo "$BODY" | jq '.'
        echo "$BODY"
    else
        echo "⚠️  NON-JSON RESPONSE (showing first 30 lines):"
        echo "$BODY" | head -30
        echo ""
        echo "This indicates the API returned HTML instead of JSON."
        echo "Common causes:"
        echo "  1. Middleware rejecting the request"
        echo "  2. Route not found (404)"
        echo "  3. Panic/error recovery returning HTML"
        echo "  4. CORS or authentication issues"
    fi
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

print_section "DIAGNOSTIC TESTS"

# Test 1: Login
print_info "Test: Superadmin Login"
make_request "POST" "$BASE_URL/login" "" "{\"email\": \"$SUPER_ADMIN_EMAIL\", \"password\": \"$SUPER_ADMIN_PASSWORD\"}"
SUPER_ADMIN_TOKEN=$(echo "$BODY" | jq -r '.data.access_token // empty')

if [ -z "$SUPER_ADMIN_TOKEN" ]; then
    echo -e "${RED}FATAL: Cannot get superadmin token. Exiting.${NC}"
    exit 1
fi

echo -e "\n${GREEN}✓ Got superadmin token${NC}"

# Test 2: List admins to see current state
print_info "Test: List All Admins"
make_request "GET" "$BASE_URL/admins" "$SUPER_ADMIN_TOKEN"

# Test 3: Create a test admin
print_info "Test: Create Admin"
TEST_EMAIL="debug_$(date +%s)@example.com"
make_request "POST" "$BASE_URL/admins" "$SUPER_ADMIN_TOKEN" "{\"email\": \"$TEST_EMAIL\", \"name\": \"Debug Admin\", \"role\": \"admin\"}"

TARGET_ADMIN_ID=$(echo "$BODY" | jq -r '.data.user.id // empty')

if [ -z "$TARGET_ADMIN_ID" ]; then
    echo -e "${RED}FATAL: Cannot create test admin. Exiting.${NC}"
    exit 1
fi

echo -e "\n${GREEN}✓ Created test admin: $TARGET_ADMIN_ID${NC}"

# Test 4: Attempt to change to same role (should fail)
print_info "Test: Change to Same Role (should fail with 'already has this role')"
make_request "POST" "$BASE_URL/admins/$TARGET_ADMIN_ID/role/initiate" "$SUPER_ADMIN_TOKEN" '{"new_role": "admin"}'

# Test 5: Attempt promotion without confirmation (should fail with 412)
print_info "Test: Promote to Superadmin Without Confirmation"
make_request "POST" "$BASE_URL/admins/$TARGET_ADMIN_ID/role/initiate" "$SUPER_ADMIN_TOKEN" '{"new_role": "superadmin"}'

# Test 6: Attempt promotion with false confirmation (should fail with 412)
print_info "Test: Promote to Superadmin With confirm_superadmin=false"
make_request "POST" "$BASE_URL/admins/$TARGET_ADMIN_ID/role/initiate" "$SUPER_ADMIN_TOKEN" '{"new_role": "superadmin", "confirm_superadmin": false}'

# Test 7: Valid promotion with confirmation
print_info "Test: Promote to Superadmin With confirm_superadmin=true (SHOULD SUCCEED)"
make_request "POST" "$BASE_URL/admins/$TARGET_ADMIN_ID/role/initiate" "$SUPER_ADMIN_TOKEN" '{"new_role": "superadmin", "confirm_superadmin": true}'

CONFIRMATION_TOKEN=$(echo "$BODY" | jq -r '.data.token // empty')

if [ ! -z "$CONFIRMATION_TOKEN" ]; then
    echo -e "\n${GREEN}✓ Got confirmation token: ${CONFIRMATION_TOKEN:0:20}...${NC}"
    
    # Test 8: Confirm the role change
    print_info "Test: Confirm Role Change"
    make_request "POST" "$BASE_URL/admins/role/confirm" "$SUPER_ADMIN_TOKEN" "{\"confirmation_token\": \"$CONFIRMATION_TOKEN\"}"
    
    # Test 9: Try to reuse token
    print_info "Test: Reuse Token (should fail)"
    make_request "POST" "$BASE_URL/admins/role/confirm" "$SUPER_ADMIN_TOKEN" "{\"confirmation_token\": \"$CONFIRMATION_TOKEN\"}"
    
    # Test 10: Check audit logs
    print_info "Test: Get Audit Logs"
    make_request "GET" "$BASE_URL/admins/audit-logs" "$SUPER_ADMIN_TOKEN"
    
    # Test 11: Get audit logs for specific admin
    print_info "Test: Get Audit Logs for Target Admin"
    make_request "GET" "$BASE_URL/admins/audit-logs?target_id=$TARGET_ADMIN_ID" "$SUPER_ADMIN_TOKEN"
else
    echo -e "\n${RED}✗ Did not get confirmation token - role change initiation failed${NC}"
    echo "This is the main issue to investigate."
fi

# Test 12: Invalid token confirmation
print_info "Test: Invalid Token"
make_request "POST" "$BASE_URL/admins/role/confirm" "$SUPER_ADMIN_TOKEN" '{"confirmation_token": "fake-token-12345"}'

# Test 13: Test with invalid admin ID
print_info "Test: Invalid Admin ID Format"
make_request "POST" "$BASE_URL/admins/not-a-uuid/role/initiate" "$SUPER_ADMIN_TOKEN" '{"new_role": "admin"}'

# Test 14: Test without auth token
print_info "Test: No Authorization (should fail)"
curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$BASE_URL/admins/audit-logs" \
  -H "Content-Type: application/json" | head -20

print_section "SUMMARY & NEXT STEPS"

echo "Review the responses above to identify:"
echo "1. Are you getting HTML responses instead of JSON?"
echo "2. What HTTP status codes are being returned?"
echo "3. Are the error messages what you expect?"
echo ""
echo "Common issues to check:"
echo "  - Is RequireSuperAdmin() middleware working correctly?"
echo "  - Are routes properly registered?"
echo "  - Is the database connection working?"
echo "  - Are there any panics in the server logs?"
echo ""
echo "Database checks to run:"
echo "  SELECT * FROM admins WHERE id = '$TARGET_ADMIN_ID';"
echo "  SELECT * FROM role_change_confirmations ORDER BY created_at DESC LIMIT 3;"
echo "  SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT 3;"
echo "  SELECT * FROM access_tokens WHERE owner_id = '$TARGET_ADMIN_ID';"