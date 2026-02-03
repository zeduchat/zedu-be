#!/bin/bash

# User Growth Endpoint Test Suite
# Tests the /api/v1/backoffice/dashboard/user-growth endpoint

set -e

# Configuration
BASE_URL="http://localhost:8019"
API_VERSION="api/v1"
ENDPOINT="${BASE_URL}/${API_VERSION}/backoffice/dashboard/user-growth"
LOGIN_ENDPOINT="${BASE_URL}/${API_VERSION}/backoffice/login"

# Credentials
SUPER_ADMIN_EMAIL="superadmin@example.com"
SUPER_ADMIN_PASSWORD="superadmin12345"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color
BLUE='\033[0;34m'

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to print section headers
print_section() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

# Function to print test results
print_result() {
    local test_name=$1
    local status=$2
    local message=$3
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if [ "$status" == "PASS" ]; then
        echo -e "${GREEN}✓ PASS${NC}: $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ FAIL${NC}: $test_name"
        echo -e "  ${YELLOW}Reason: $message${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# Function to make authenticated request
make_request() {
    local method=$1
    local url=$2
    local expected_status=$3
    
    response=$(curl -s -w "\n%{http_code}" -X "$method" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        "$url")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    echo "$http_code|$body"
}

# Login and get token
print_section "AUTHENTICATION"
echo "Logging in as superadmin..."

login_response=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$SUPER_ADMIN_EMAIL\",\"password\":\"$SUPER_ADMIN_PASSWORD\"}" \
    "$LOGIN_ENDPOINT")

login_http_code=$(echo "$login_response" | tail -n1)
login_body=$(echo "$login_response" | sed '$d')

if [ "$login_http_code" != "200" ]; then
    echo -e "${RED}Failed to login. HTTP Code: $login_http_code${NC}"
    echo "Response: $login_body"
    exit 1
fi

TOKEN=$(echo "$login_body" | grep -o '"access_token":"[^"]*' | sed 's/"access_token":"//')

if [ -z "$TOKEN" ]; then
    echo -e "${RED}Failed to extract token from login response${NC}"
    echo "Response: $login_body"
    exit 1
fi

echo -e "${GREEN}✓ Successfully authenticated${NC}"
echo "Token: ${TOKEN:0:20}..."

# Test 1: Valid preset - today
print_section "TEST GROUP 1: VALID PRESET REQUESTS"

result=$(make_request "GET" "${ENDPOINT}?preset=today" "200")
http_code=$(echo "$result" | cut -d'|' -f1)
body=$(echo "$result" | cut -d'|' -f2-)

if [ "$http_code" == "200" ] && echo "$body" | grep -q "total_count"; then
    print_result "Preset: today" "PASS"
else
    print_result "Preset: today" "FAIL" "HTTP $http_code or missing total_count"
fi

# Test 2: Valid preset - yesterday
result=$(make_request "GET" "${ENDPOINT}?preset=yesterday" "200")
http_code=$(echo "$result" | cut -d'|' -f1)
body=$(echo "$result" | cut -d'|' -f2-)

if [ "$http_code" == "200" ]; then
    print_result "Preset: yesterday" "PASS"
else
    print_result "Preset: yesterday" "FAIL" "HTTP $http_code"
fi

# Test 3: Valid preset - last_7_days
result=$(make_request "GET" "${ENDPOINT}?preset=last_7_days" "200")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "200" ]; then
    print_result "Preset: last_7_days" "PASS"
else
    print_result "Preset: last_7_days" "FAIL" "HTTP $http_code"
fi

# Test 4: Valid preset - last_30_days
result=$(make_request "GET" "${ENDPOINT}?preset=last_30_days" "200")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "200" ]; then
    print_result "Preset: last_30_days" "PASS"
else
    print_result "Preset: last_30_days" "FAIL" "HTTP $http_code"
fi

# Test 5: Valid preset - this_month
result=$(make_request "GET" "${ENDPOINT}?preset=this_month" "200")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "200" ]; then
    print_result "Preset: this_month" "PASS"
else
    print_result "Preset: this_month" "FAIL" "HTTP $http_code"
fi

# Test 6: Valid preset - this_year
result=$(make_request "GET" "${ENDPOINT}?preset=this_year" "200")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "200" ]; then
    print_result "Preset: this_year" "PASS"
else
    print_result "Preset: this_year" "FAIL" "HTTP $http_code"
fi

# Test 7: Valid custom date range
print_section "TEST GROUP 2: CUSTOM DATE RANGES"

result=$(make_request "GET" "${ENDPOINT}?from=2024-01-01&to=2024-01-31" "200")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "200" ]; then
    print_result "Custom date range (valid)" "PASS"
else
    print_result "Custom date range (valid)" "FAIL" "HTTP $http_code"
fi

# Test 8: Same from and to date
result=$(make_request "GET" "${ENDPOINT}?from=2024-01-15&to=2024-01-15" "200")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "200" ]; then
    print_result "Same from and to date" "PASS"
else
    print_result "Same from and to date" "FAIL" "HTTP $http_code"
fi

# Test 9: Date range exactly 2 years (boundary test)
result=$(make_request "GET" "${ENDPOINT}?from=2022-01-01&to=2024-01-01" "200")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "200" ]; then
    print_result "Date range exactly 2 years" "PASS"
else
    print_result "Date range exactly 2 years" "FAIL" "HTTP $http_code"
fi

# Test 10: GROUP BY tests
print_section "TEST GROUP 3: GROUP BY VARIATIONS"

result=$(make_request "GET" "${ENDPOINT}?preset=last_7_days&group_by=day" "200")
http_code=$(echo "$result" | cut -d'|' -f1)
body=$(echo "$result" | cut -d'|' -f2-)

if [ "$http_code" == "200" ] && echo "$body" | grep -q "breakdown"; then
    print_result "Group by day" "PASS"
else
    print_result "Group by day" "FAIL" "HTTP $http_code or missing breakdown"
fi

result=$(make_request "GET" "${ENDPOINT}?preset=last_30_days&group_by=week" "200")
http_code=$(echo "$result" | cut -d'|' -f1)
body=$(echo "$result" | cut -d'|' -f2-)

if [ "$http_code" == "200" ] && echo "$body" | grep -q "breakdown"; then
    print_result "Group by week" "PASS"
else
    print_result "Group by week" "FAIL" "HTTP $http_code or missing breakdown"
fi

result=$(make_request "GET" "${ENDPOINT}?preset=this_year&group_by=month" "200")
http_code=$(echo "$result" | cut -d'|' -f1)
body=$(echo "$result" | cut -d'|' -f2-)

if [ "$http_code" == "200" ] && echo "$body" | grep -q "breakdown"; then
    print_result "Group by month" "PASS"
else
    print_result "Group by month" "FAIL" "HTTP $http_code or missing breakdown"
fi

# Test 11: Custom date range with group_by
result=$(make_request "GET" "${ENDPOINT}?from=2024-01-01&to=2024-01-31&group_by=day" "200")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "200" ]; then
    print_result "Custom date range with group_by" "PASS"
else
    print_result "Custom date range with group_by" "FAIL" "HTTP $http_code"
fi

# Test 12: TIMEZONE tests
print_section "TEST GROUP 4: TIMEZONE HANDLING"

result=$(make_request "GET" "${ENDPOINT}?preset=today&timezone=America/New_York" "200")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "200" ]; then
    print_result "Timezone: America/New_York" "PASS"
else
    print_result "Timezone: America/New_York" "FAIL" "HTTP $http_code"
fi

result=$(make_request "GET" "${ENDPOINT}?preset=today&timezone=Europe/London" "200")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "200" ]; then
    print_result "Timezone: Europe/London" "PASS"
else
    print_result "Timezone: Europe/London" "FAIL" "HTTP $http_code"
fi

result=$(make_request "GET" "${ENDPOINT}?preset=today&timezone=Asia/Tokyo" "200")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "200" ]; then
    print_result "Timezone: Asia/Tokyo" "PASS"
else
    print_result "Timezone: Asia/Tokyo" "FAIL" "HTTP $http_code"
fi

# Test 13: ERROR CASES - Invalid preset
print_section "TEST GROUP 5: ERROR HANDLING - INVALID INPUTS"

result=$(make_request "GET" "${ENDPOINT}?preset=invalid_preset" "400")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "400" ]; then
    print_result "Invalid preset value" "PASS"
else
    print_result "Invalid preset value" "FAIL" "Expected 400, got $http_code"
fi

# Test 14: Invalid group_by
result=$(make_request "GET" "${ENDPOINT}?preset=today&group_by=invalid" "400")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "400" ]; then
    print_result "Invalid group_by value" "PASS"
else
    print_result "Invalid group_by value" "FAIL" "Expected 400, got $http_code"
fi

# Test 15: Invalid date format
result=$(make_request "GET" "${ENDPOINT}?from=2024/01/01&to=2024-01-31" "400")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "400" ]; then
    print_result "Invalid from date format" "PASS"
else
    print_result "Invalid from date format" "FAIL" "Expected 400, got $http_code"
fi

result=$(make_request "GET" "${ENDPOINT}?from=2024-01-01&to=31-01-2024" "400")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "400" ]; then
    print_result "Invalid to date format" "PASS"
else
    print_result "Invalid to date format" "FAIL" "Expected 400, got $http_code"
fi

# Test 16: Missing parameters
result=$(make_request "GET" "${ENDPOINT}" "400")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "400" ]; then
    print_result "Missing all parameters" "PASS"
else
    print_result "Missing all parameters" "FAIL" "Expected 400, got $http_code"
fi

# Test 17: Only from date provided
result=$(make_request "GET" "${ENDPOINT}?from=2024-01-01" "400")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "400" ]; then
    print_result "Only from date provided" "PASS"
else
    print_result "Only from date provided" "FAIL" "Expected 400, got $http_code"
fi

# Test 18: Only to date provided
result=$(make_request "GET" "${ENDPOINT}?to=2024-01-31" "400")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "400" ]; then
    print_result "Only to date provided" "PASS"
else
    print_result "Only to date provided" "FAIL" "Expected 400, got $http_code"
fi

# Test 19: Both preset and custom dates (should fail)
result=$(make_request "GET" "${ENDPOINT}?preset=today&from=2024-01-01&to=2024-01-31" "400")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "400" ]; then
    print_result "Preset with custom dates (conflict)" "PASS"
else
    print_result "Preset with custom dates (conflict)" "FAIL" "Expected 400, got $http_code"
fi

# Test 20: To date before from date
result=$(make_request "GET" "${ENDPOINT}?from=2024-01-31&to=2024-01-01" "400")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "400" ]; then
    print_result "To date before from date" "PASS"
else
    print_result "To date before from date" "FAIL" "Expected 400, got $http_code"
fi

# Test 21: Date range exceeds 2 years
result=$(make_request "GET" "${ENDPOINT}?from=2020-01-01&to=2024-01-01" "400")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "400" ]; then
    print_result "Date range exceeds 2 years" "PASS"
else
    print_result "Date range exceeds 2 years" "FAIL" "Expected 400, got $http_code"
fi

# Test 22: Invalid timezone
result=$(make_request "GET" "${ENDPOINT}?preset=today&timezone=Invalid/Timezone" "500")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "400" ] || [ "$http_code" == "500" ]; then
    print_result "Invalid timezone" "PASS"
else
    print_result "Invalid timezone" "FAIL" "Expected 400/500, got $http_code"
fi

# Test 23: EDGE CASES
print_section "TEST GROUP 6: EDGE CASES"

# Leap year date
result=$(make_request "GET" "${ENDPOINT}?from=2024-02-29&to=2024-02-29" "200")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "200" ]; then
    print_result "Leap year date (Feb 29, 2024)" "PASS"
else
    print_result "Leap year date (Feb 29, 2024)" "FAIL" "HTTP $http_code"
fi

# Month boundary
result=$(make_request "GET" "${ENDPOINT}?from=2024-01-31&to=2024-02-01" "200")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "200" ]; then
    print_result "Month boundary crossing" "PASS"
else
    print_result "Month boundary crossing" "FAIL" "HTTP $http_code"
fi

# Year boundary
result=$(make_request "GET" "${ENDPOINT}?from=2023-12-31&to=2024-01-01" "200")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "200" ]; then
    print_result "Year boundary crossing" "PASS"
else
    print_result "Year boundary crossing" "FAIL" "HTTP $http_code"
fi

# Very long date range (but within 2 years)
result=$(make_request "GET" "${ENDPOINT}?from=2023-01-01&to=2024-12-31&group_by=month" "200")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "200" ]; then
    print_result "Long date range with monthly grouping" "PASS"
else
    print_result "Long date range with monthly grouping" "FAIL" "HTTP $http_code"
fi

# Single day with daily breakdown
result=$(make_request "GET" "${ENDPOINT}?from=2024-01-15&to=2024-01-15&group_by=day" "200")
http_code=$(echo "$result" | cut -d'|' -f1)

if [ "$http_code" == "200" ]; then
    print_result "Single day with daily breakdown" "PASS"
else
    print_result "Single day with daily breakdown" "FAIL" "HTTP $http_code"
fi

# Test 24: RESPONSE STRUCTURE VALIDATION
print_section "TEST GROUP 7: RESPONSE STRUCTURE VALIDATION"

result=$(make_request "GET" "${ENDPOINT}?preset=last_7_days&group_by=day" "200")
http_code=$(echo "$result" | cut -d'|' -f1)
body=$(echo "$result" | cut -d'|' -f2-)

if echo "$body" | grep -q '"start_date"' && \
   echo "$body" | grep -q '"end_date"' && \
   echo "$body" | grep -q '"total_count"' && \
   echo "$body" | grep -q '"breakdown"' && \
   echo "$body" | grep -q '"period"'; then
    print_result "Response contains all required fields" "PASS"
else
    print_result "Response contains all required fields" "FAIL" "Missing required fields"
fi

# Test 25: Check breakdown array structure
if echo "$body" | grep -q '"date"' && \
   echo "$body" | grep -q '"count"'; then
    print_result "Breakdown contains required fields" "PASS"
else
    print_result "Breakdown contains required fields" "FAIL" "Missing breakdown fields"
fi

# Test 26: AUTHORIZATION TEST (if possible to test without token)
print_section "TEST GROUP 8: AUTHORIZATION"

no_auth_result=$(curl -s -w "\n%{http_code}" -X GET "${ENDPOINT}?preset=today")
http_code=$(echo "$no_auth_result" | tail -n1)

if [ "$http_code" == "401" ] || [ "$http_code" == "403" ]; then
    print_result "Unauthorized access blocked" "PASS"
else
    print_result "Unauthorized access blocked" "FAIL" "Expected 401/403, got $http_code"
fi

# FINAL SUMMARY
print_section "TEST SUMMARY"
echo -e "Total Tests: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All tests passed!${NC}\n"
    exit 0
else
    echo -e "\n${RED}⚠ Some tests failed. Please review the output above.${NC}\n"
    exit 1
fi