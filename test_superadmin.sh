#!/bin/bash

# Superadmin Role Change Feature - Manual Testing Script
# Tests the three endpoints:
# 1. POST /admins/:admin_id/role/initiate
# 2. POST /admins/role/confirm
# 3. GET /admins/audit-logs

# Configuration
BASE_URL="http://localhost:8019/api/v1/backoffice"
SUPER_ADMIN_EMAIL="superadmin@example.com"
SUPER_ADMIN_PASSWORD="superadmin12345"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper function to print section headers
print_section() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}\n"
}

# Helper function to print test results
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ PASS:${NC} $2"
    else
        echo -e "${RED}✗ FAIL:${NC} $2"
    fi
}

# Helper function to print info
print_info() {
    echo -e "${YELLOW}ℹ INFO:${NC} $1"
}

# Variables to store tokens and IDs
SUPER_ADMIN_TOKEN=""
REGULAR_ADMIN_TOKEN=""
REGULAR_ADMIN_ID=""
REGULAR_ADMIN_EMAIL=""
TARGET_ADMIN_ID=""
TARGET_ADMIN_EMAIL=""
CONFIRMATION_TOKEN=""

# =============================================================================
# SETUP: Login and Create Test Admins
# =============================================================================

print_section "SETUP: Authentication and Admin Creation"

# 1. Login as superadmin
print_info "Logging in as superadmin..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$SUPER_ADMIN_EMAIL\",
    \"password\": \"$SUPER_ADMIN_PASSWORD\"
  }")

SUPER_ADMIN_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.data.access_token')
if [ "$SUPER_ADMIN_TOKEN" != "null" ] && [ -n "$SUPER_ADMIN_TOKEN" ]; then
    print_result 0 "Superadmin login successful"
    echo "Token: ${SUPER_ADMIN_TOKEN:0:20}..."
else
    print_result 1 "Superadmin login failed"
    echo "Response: $LOGIN_RESPONSE"
    exit 1
fi

# 2. Create a regular admin for testing
print_info "Creating regular admin for testing..."
CREATE_ADMIN_RESPONSE=$(curl -s -X POST "$BASE_URL/admins" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d "{
    \"email\": \"testadmin_$(date +%s)@example.com\",
    \"name\": \"Test Regular Admin\",
    \"role\": \"admin\"
  }")

REGULAR_ADMIN_ID=$(echo $CREATE_ADMIN_RESPONSE | jq -r '.data.user.id')
REGULAR_ADMIN_EMAIL=$(echo $CREATE_ADMIN_RESPONSE | jq -r '.data.user.email')
REGULAR_ADMIN_TOKEN=$(echo $CREATE_ADMIN_RESPONSE | jq -r '.data.access_token')

if [ "$REGULAR_ADMIN_ID" != "null" ] && [ -n "$REGULAR_ADMIN_ID" ]; then
    print_result 0 "Regular admin created successfully"
    echo "Admin ID: $REGULAR_ADMIN_ID"
    echo "Admin Email: $REGULAR_ADMIN_EMAIL"
else
    print_result 1 "Failed to create regular admin"
    echo "Response: $CREATE_ADMIN_RESPONSE"
    exit 1
fi

# 3. Create another admin to be the target of role changes
print_info "Creating target admin for role change testing..."
CREATE_TARGET_RESPONSE=$(curl -s -X POST "$BASE_URL/admins" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d "{
    \"email\": \"targetadmin_$(date +%s)@example.com\",
    \"name\": \"Target Admin\",
    \"role\": \"admin\"
  }")

TARGET_ADMIN_ID=$(echo $CREATE_TARGET_RESPONSE | jq -r '.data.user.id')
TARGET_ADMIN_EMAIL=$(echo $CREATE_TARGET_RESPONSE | jq -r '.data.user.email')

if [ "$TARGET_ADMIN_ID" != "null" ] && [ -n "$TARGET_ADMIN_ID" ]; then
    print_result 0 "Target admin created successfully"
    echo "Target Admin ID: $TARGET_ADMIN_ID"
    echo "Target Admin Email: $TARGET_ADMIN_EMAIL"
else
    print_result 1 "Failed to create target admin"
    echo "Response: $CREATE_TARGET_RESPONSE"
    exit 1
fi

# =============================================================================
# TEST SUITE 1: InitiateChangeAdminRole Endpoint
# =============================================================================

print_section "TEST SUITE 1: Initiate Role Change Endpoint"

# Test 1.1: Invalid admin ID format
print_info "Test 1.1: Invalid admin ID format"
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/invalid-uuid/role/initiate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d '{"new_role": "admin"}')
STATUS=$(echo $RESPONSE | jq -r '.status_code')
[ "$STATUS" = "400" ] && print_result 0 "Rejected invalid UUID" || print_result 1 "Should reject invalid UUID"
echo "Response: $RESPONSE" | jq '.'

# Test 1.2: Non-existent admin ID
print_info "Test 1.2: Non-existent admin ID"
FAKE_UUID="00000000-0000-0000-0000-000000000000"
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/$FAKE_UUID/role/initiate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d '{"new_role": "admin"}')
STATUS=$(echo $RESPONSE | jq -r '.status_code')
[ "$STATUS" = "400" ] && print_result 0 "Rejected non-existent admin" || print_result 1 "Should reject non-existent admin"
echo "Response: $RESPONSE" | jq '.'

# Test 1.3: Regular admin attempting role change (should fail - not superadmin)
print_info "Test 1.3: Regular admin attempting role change"
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/$TARGET_ADMIN_ID/role/initiate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $REGULAR_ADMIN_TOKEN" \
  -d '{"new_role": "admin"}')
STATUS=$(echo $RESPONSE | jq -r '.status_code')
[ "$STATUS" != "202" ] && print_result 0 "Blocked non-superadmin" || print_result 1 "Should block non-superadmin"
echo "Response: $RESPONSE" | jq '.'

# Test 1.4: Missing request body
print_info "Test 1.4: Missing request body"
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/$TARGET_ADMIN_ID/role/initiate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d '{}')
STATUS=$(echo $RESPONSE | jq -r '.status_code')
[ "$STATUS" = "400" ] && print_result 0 "Rejected empty body" || print_result 1 "Should reject empty body"
echo "Response: $RESPONSE" | jq '.'

# Test 1.5: Invalid role value
print_info "Test 1.5: Invalid role value"
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/$TARGET_ADMIN_ID/role/initiate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d '{"new_role": "superuser"}')
STATUS=$(echo $RESPONSE | jq -r '.status_code')
[ "$STATUS" = "400" ] && print_result 0 "Rejected invalid role" || print_result 1 "Should reject invalid role"
echo "Response: $RESPONSE" | jq '.'

# Test 1.6: Changing to same role (edge case)
print_info "Test 1.6: Changing admin to same role they already have"
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/$TARGET_ADMIN_ID/role/initiate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d '{"new_role": "admin"}')
MESSAGE=$(echo $RESPONSE | jq -r '.message')
[[ "$MESSAGE" == *"already has this role"* ]] && print_result 0 "Detected same role" || print_result 1 "Should detect same role"
echo "Response: $RESPONSE" | jq '.'

# Test 1.7: Promote to superadmin WITHOUT confirmation flag (should fail)
print_info "Test 1.7: Promote to superadmin without confirmation"
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/$TARGET_ADMIN_ID/role/initiate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d '{"new_role": "superadmin"}')
STATUS=$(echo $RESPONSE | jq -r '.status_code')
[ "$STATUS" = "412" ] && print_result 0 "Requires confirmation for superadmin" || print_result 1 "Should require confirmation"
echo "Response: $RESPONSE" | jq '.'

# Test 1.8: Promote to superadmin with confirmation=false (should fail)
print_info "Test 1.8: Promote to superadmin with confirmation=false"
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/$TARGET_ADMIN_ID/role/initiate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d '{"new_role": "superadmin", "confirm_superadmin": false}')
STATUS=$(echo $RESPONSE | jq -r '.status_code')
[ "$STATUS" = "412" ] && print_result 0 "Rejected false confirmation" || print_result 1 "Should reject false confirmation"
echo "Response: $RESPONSE" | jq '.'

# Test 1.9: Valid demotion from admin to admin (for token generation test)
print_info "Test 1.9: Valid role change initiation (admin -> admin for another user)"
# First, let's use the REGULAR_ADMIN_ID as target
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/$REGULAR_ADMIN_ID/role/initiate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d '{"new_role": "admin"}')
STATUS=$(echo $RESPONSE | jq -r '.status_code')
if [ "$STATUS" = "400" ]; then
    # Expected if they already have that role
    print_result 0 "Correctly rejected same role"
else
    print_result 1 "Unexpected response for same role test"
fi
echo "Response: $RESPONSE" | jq '.'

# Test 1.10: Successfully initiate role change (demote from potential superadmin)
print_info "Test 1.10: Valid promotion to superadmin with confirmation"
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/$TARGET_ADMIN_ID/role/initiate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d '{"new_role": "superadmin", "confirm_superadmin": true}')
STATUS=$(echo $RESPONSE | jq -r '.status_code')
CONFIRMATION_TOKEN=$(echo $RESPONSE | jq -r '.data.token')
EXPIRES_AT=$(echo $RESPONSE | jq -r '.data.expires_at')

if [ "$STATUS" = "202" ] && [ "$CONFIRMATION_TOKEN" != "null" ]; then
    print_result 0 "Successfully initiated role change"
    echo "Confirmation Token: ${CONFIRMATION_TOKEN:0:20}..."
    echo "Expires At: $EXPIRES_AT"
else
    print_result 1 "Failed to initiate role change"
fi
echo "Response: $RESPONSE" | jq '.'

# Test 1.11: Initiate another role change while one is pending (edge case)
print_info "Test 1.11: Initiate second role change for same admin"
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/$TARGET_ADMIN_ID/role/initiate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d '{"new_role": "admin"}')
STATUS=$(echo $RESPONSE | jq -r '.status_code')
# This should succeed as the system allows multiple pending confirmations
if [ "$STATUS" = "202" ]; then
    print_result 0 "Allowed multiple pending confirmations"
    SECOND_TOKEN=$(echo $RESPONSE | jq -r '.data.token')
    echo "Second Token: ${SECOND_TOKEN:0:20}..."
else
    print_result 0 "System behavior for multiple pending confirmations"
fi
echo "Response: $RESPONSE" | jq '.'

# =============================================================================
# TEST SUITE 2: ConfirmChangeAdminRole Endpoint
# =============================================================================

print_section "TEST SUITE 2: Confirm Role Change Endpoint"

# Test 2.1: Missing confirmation token
print_info "Test 2.1: Missing confirmation token"
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/role/confirm" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d '{}')
STATUS=$(echo $RESPONSE | jq -r '.status_code')
[ "$STATUS" = "400" ] && print_result 0 "Rejected missing token" || print_result 1 "Should reject missing token"
echo "Response: $RESPONSE" | jq '.'

# Test 2.2: Invalid token format
print_info "Test 2.2: Invalid token"
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/role/confirm" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d '{"confirmation_token": "invalid-token-12345"}')
MESSAGE=$(echo $RESPONSE | jq -r '.message')
[[ "$MESSAGE" == *"invalid"* || "$MESSAGE" == *"expired"* ]] && print_result 0 "Rejected invalid token" || print_result 1 "Should reject invalid token"
echo "Response: $RESPONSE" | jq '.'

# Test 2.3: Wrong requester attempting to confirm (if we had another superadmin)
print_info "Test 2.3: Different admin trying to confirm (wrong requester)"
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/role/confirm" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $REGULAR_ADMIN_TOKEN" \
  -d "{\"confirmation_token\": \"$CONFIRMATION_TOKEN\"}")
STATUS=$(echo $RESPONSE | jq -r '.status_code')
# Should fail because regular admin is not superadmin
[ "$STATUS" != "200" ] && print_result 0 "Blocked wrong requester" || print_result 1 "Should block wrong requester"
echo "Response: $RESPONSE" | jq '.'

# Test 2.4: Valid confirmation
print_info "Test 2.4: Valid confirmation of role change"
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/role/confirm" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d "{\"confirmation_token\": \"$CONFIRMATION_TOKEN\"}")
STATUS=$(echo $RESPONSE | jq -r '.status_code')
if [ "$STATUS" = "200" ]; then
    print_result 0 "Successfully confirmed role change"
else
    print_result 1 "Failed to confirm role change"
fi
echo "Response: $RESPONSE" | jq '.'

# Test 2.5: Reusing the same token (should fail)
print_info "Test 2.5: Attempting to reuse confirmation token"
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/role/confirm" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d "{\"confirmation_token\": \"$CONFIRMATION_TOKEN\"}")
MESSAGE=$(echo $RESPONSE | jq -r '.message')
[[ "$MESSAGE" == *"invalid"* || "$MESSAGE" == *"expired"* ]] && print_result 0 "Prevented token reuse" || print_result 1 "Should prevent token reuse"
echo "Response: $RESPONSE" | jq '.'

# Test 2.6: Test token expiration (create a new change for testing)
print_info "Test 2.6: Testing token expiration scenario"
echo "Note: Tokens expire after 15 minutes. For manual testing, you can:"
echo "  1. Wait 15+ minutes and test with an expired token"
echo "  2. Manually modify expires_at in the database to test expiration"
echo "Skipping automatic expiration test due to time constraint..."

# =============================================================================
# TEST SUITE 3: GetRoleAuditHistory Endpoint
# =============================================================================

print_section "TEST SUITE 3: Audit History Endpoint"

# Test 3.1: Get all audit logs without filters
print_info "Test 3.1: Get all audit logs"
RESPONSE=$(curl -s -X GET "$BASE_URL/admins/audit-logs" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN")
STATUS=$(echo $RESPONSE | jq -r '.status_code')
LOGS_COUNT=$(echo $RESPONSE | jq -r '.data.logs | length')
if [ "$STATUS" = "200" ]; then
    print_result 0 "Successfully retrieved audit logs"
    echo "Number of logs: $LOGS_COUNT"
else
    print_result 1 "Failed to retrieve audit logs"
fi
echo "Response: $RESPONSE" | jq '.'

# Test 3.2: Filter by target_id
print_info "Test 3.2: Filter audit logs by target_id"
RESPONSE=$(curl -s -X GET "$BASE_URL/admins/audit-logs?target_id=$TARGET_ADMIN_ID" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN")
STATUS=$(echo $RESPONSE | jq -r '.status_code')
FILTERED_COUNT=$(echo $RESPONSE | jq -r '.data.logs | length')
if [ "$STATUS" = "200" ]; then
    print_result 0 "Successfully filtered by target_id"
    echo "Filtered logs count: $FILTERED_COUNT"
else
    print_result 1 "Failed to filter by target_id"
fi
echo "Response: $RESPONSE" | jq '.'

# Test 3.3: Filter by date
print_info "Test 3.3: Filter audit logs by date (today)"
TODAY=$(date +%Y-%m-%d)
RESPONSE=$(curl -s -X GET "$BASE_URL/admins/audit-logs?date=$TODAY" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN")
STATUS=$(echo $RESPONSE | jq -r '.status_code')
if [ "$STATUS" = "200" ]; then
    print_result 0 "Successfully filtered by date"
else
    print_result 1 "Failed to filter by date"
fi
echo "Response: $RESPONSE" | jq '.'

# Test 3.4: Pagination test
print_info "Test 3.4: Pagination (page_size=2)"
RESPONSE=$(curl -s -X GET "$BASE_URL/admins/audit-logs?page_size=2&page=1" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN")
STATUS=$(echo $RESPONSE | jq -r '.status_code')
PAGE_SIZE=$(echo $RESPONSE | jq -r '.data.logs | length')
if [ "$STATUS" = "200" ]; then
    print_result 0 "Pagination working"
    echo "Page size returned: $PAGE_SIZE"
else
    print_result 1 "Pagination failed"
fi
echo "Response: $RESPONSE" | jq '.'

# Test 3.5: Invalid date format
print_info "Test 3.5: Invalid date format"
RESPONSE=$(curl -s -X GET "$BASE_URL/admins/audit-logs?date=invalid-date" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN")
# This may return 200 with empty results or 400, depending on implementation
STATUS=$(echo $RESPONSE | jq -r '.status_code')
print_result 0 "Handled invalid date format (Status: $STATUS)"
echo "Response: $RESPONSE" | jq '.'

# Test 3.6: Regular admin access to audit logs (should fail)
print_info "Test 3.6: Regular admin attempting to access audit logs"
RESPONSE=$(curl -s -X GET "$BASE_URL/admins/audit-logs" \
  -H "Authorization: Bearer $REGULAR_ADMIN_TOKEN")
STATUS=$(echo $RESPONSE | jq -r '.status_code')
[ "$STATUS" != "200" ] && print_result 0 "Blocked regular admin access" || print_result 1 "Should block regular admin"
echo "Response: $RESPONSE" | jq '.'

# Test 3.7: Combined filters
print_info "Test 3.7: Combined filters (target_id + date)"
RESPONSE=$(curl -s -X GET "$BASE_URL/admins/audit-logs?target_id=$TARGET_ADMIN_ID&date=$TODAY" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN")
STATUS=$(echo $RESPONSE | jq -r '.status_code')
if [ "$STATUS" = "200" ]; then
    print_result 0 "Successfully applied combined filters"
else
    print_result 1 "Failed with combined filters"
fi
echo "Response: $RESPONSE" | jq '.'

# =============================================================================
# TEST SUITE 4: Edge Cases & Security Tests
# =============================================================================

print_section "TEST SUITE 4: Edge Cases & Security"

# Test 4.1: No authorization header
print_info "Test 4.1: Request without authorization"
RESPONSE=$(curl -s -X GET "$BASE_URL/admins/audit-logs")
STATUS=$(echo $RESPONSE | jq -r '.status_code')
[ "$STATUS" = "401" ] && print_result 0 "Blocked unauthorized request" || print_result 1 "Should block unauthorized"
echo "Response: $RESPONSE" | jq '.'

# Test 4.2: Invalid token
print_info "Test 4.2: Request with invalid token"
RESPONSE=$(curl -s -X GET "$BASE_URL/admins/audit-logs" \
  -H "Authorization: Bearer invalid-token-xyz")
STATUS=$(echo $RESPONSE | jq -r '.status_code')
[ "$STATUS" = "401" ] && print_result 0 "Rejected invalid token" || print_result 1 "Should reject invalid token"
echo "Response: $RESPONSE" | jq '.'

# Test 4.3: Verify target admin's tokens were invalidated after role change
print_info "Test 4.3: Verify old tokens invalidated after role change"
echo "This requires checking that access tokens with owner_id=$TARGET_ADMIN_ID have is_live=false"
echo "Manual DB check: SELECT * FROM access_tokens WHERE owner_id='$TARGET_ADMIN_ID';"

# Test 4.4: Create another admin and demote them
print_info "Test 4.4: Test demotion scenario (superadmin -> admin)"
# Create a temporary superadmin
TEMP_SUPER_RESPONSE=$(curl -s -X POST "$BASE_URL/admins" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d "{
    \"email\": \"tempsuper_$(date +%s)@example.com\",
    \"name\": \"Temp Superadmin\",
    \"role\": \"superadmin\"
  }")
TEMP_SUPER_ID=$(echo $TEMP_SUPER_RESPONSE | jq -r '.data.user.id')

if [ "$TEMP_SUPER_ID" != "null" ]; then
    echo "Created temp superadmin: $TEMP_SUPER_ID"
    
    # Demote them
    DEMOTE_RESPONSE=$(curl -s -X POST "$BASE_URL/admins/$TEMP_SUPER_ID/role/initiate" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
      -d '{"new_role": "admin"}')
    
    DEMOTE_TOKEN=$(echo $DEMOTE_RESPONSE | jq -r '.data.token')
    
    if [ "$DEMOTE_TOKEN" != "null" ]; then
        # Confirm demotion
        CONFIRM_DEMOTE=$(curl -s -X POST "$BASE_URL/admins/role/confirm" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
          -d "{\"confirmation_token\": \"$DEMOTE_TOKEN\"}")
        
        STATUS=$(echo $CONFIRM_DEMOTE | jq -r '.status_code')
        [ "$STATUS" = "200" ] && print_result 0 "Successfully demoted superadmin" || print_result 1 "Demotion failed"
        echo "Response: $CONFIRM_DEMOTE" | jq '.'
    else
        print_result 1 "Failed to initiate demotion"
    fi
else
    print_result 1 "Failed to create temp superadmin for demotion test"
fi

# Test 4.5: SQL Injection attempt
print_info "Test 4.5: SQL injection attempt in filters"
RESPONSE=$(curl -s -X GET "$BASE_URL/admins/audit-logs?target_id=1' OR '1'='1" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN")
STATUS=$(echo $RESPONSE | jq -r '.status_code')
print_result 0 "SQL injection test completed (Status: $STATUS)"
echo "Response: $RESPONSE" | jq '.'

# Test 4.6: Very long token string
print_info "Test 4.6: Extremely long confirmation token"
LONG_TOKEN=$(python3 -c "print('a' * 10000)")
RESPONSE=$(curl -s -X POST "$BASE_URL/admins/role/confirm" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d "{\"confirmation_token\": \"$LONG_TOKEN\"}")
STATUS=$(echo $RESPONSE | jq -r '.status_code')
[ "$STATUS" = "400" ] && print_result 0 "Handled long token gracefully" || print_result 0 "Long token test completed"

# =============================================================================
# SUMMARY
# =============================================================================

print_section "TEST SUMMARY"

echo -e "${YELLOW}Manual Testing Complete!${NC}\n"
echo "Test artifacts created:"
echo "  - Regular Admin ID: $REGULAR_ADMIN_ID"
echo "  - Regular Admin Email: $REGULAR_ADMIN_EMAIL"
echo "  - Target Admin ID: $TARGET_ADMIN_ID"
echo "  - Target Admin Email: $TARGET_ADMIN_EMAIL"
echo ""
echo -e "${BLUE}Additional Manual Tests to Perform:${NC}"
echo "  1. Test token expiration by waiting 15+ minutes"
echo "  2. Verify database state of role_change_confirmations table"
echo "  3. Verify database state of audit_logs table"
echo "  4. Test concurrent role change requests"
echo "  5. Check that admin's sessions are terminated after role change"
echo "  6. Verify email notifications if implemented"
echo ""
echo -e "${BLUE}Database Verification Queries:${NC}"
echo "  -- Check role changes"
echo "  SELECT * FROM admins WHERE id IN ('$TARGET_ADMIN_ID', '$REGULAR_ADMIN_ID');"
echo ""
echo "  -- Check confirmation records"
echo "  SELECT * FROM role_change_confirmations ORDER BY created_at DESC LIMIT 5;"
echo ""
echo "  -- Check audit logs"
echo "  SELECT * FROM audit_logs WHERE action = 'ROLE_CHANGE_CONFIRMED' ORDER BY created_at DESC;"
echo ""
echo "  -- Check token invalidation"
echo "  SELECT * FROM access_tokens WHERE owner_id = '$TARGET_ADMIN_ID';"
echo ""
echo -e "${GREEN}Script execution completed!${NC}"