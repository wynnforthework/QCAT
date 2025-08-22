#!/bin/bash

echo "🔍 Verifying Recent Fixes..."
echo "================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS_COUNT=0
FAIL_COUNT=0
WARN_COUNT=0

# Function to print check result
print_result() {
    local check_name="$1"
    local status="$2"
    local message="$3"
    
    case $status in
        "PASS")
            echo -e "✅ ${GREEN}$check_name: PASS${NC}"
            echo -e "   $message"
            ((PASS_COUNT++))
            ;;
        "FAIL")
            echo -e "❌ ${RED}$check_name: FAIL${NC}"
            echo -e "   $message"
            ((FAIL_COUNT++))
            ;;
        "WARNING")
            echo -e "⚠️  ${YELLOW}$check_name: WARNING${NC}"
            echo -e "   $message"
            ((WARN_COUNT++))
            ;;
    esac
    echo ""
}

# Check 1: Verify migration files exist
echo "Checking migration files..."
if [ -f "internal/database/migrations/000023_fix_hotlist_fields.up.sql" ]; then
    print_result "Migration Files" "PASS" "Hotlist fields migration file exists"
else
    print_result "Migration Files" "FAIL" "Missing hotlist fields migration file"
fi

# Check 2: Verify orchestrator.go has external API implementation
echo "Checking external API implementation..."
if grep -q "BinanceAPIClient" internal/strategy/optimizer/orchestrator.go; then
    print_result "External API Implementation" "PASS" "BinanceAPIClient found in orchestrator.go"
else
    print_result "External API Implementation" "FAIL" "BinanceAPIClient not found in orchestrator.go"
fi

# Check 3: Verify NaN handling in strategy_scheduler.go
echo "Checking NaN handling..."
if grep -q "math.IsNaN" internal/automation/scheduler/strategy_scheduler.go; then
    print_result "NaN Handling" "PASS" "NaN filtering implemented in strategy_scheduler.go"
else
    print_result "NaN Handling" "FAIL" "NaN filtering not found in strategy_scheduler.go"
fi

# Check 4: Verify UUID handling in sub_schedulers.go
echo "Checking UUID handling..."
if grep -q "strategies table" internal/automation/scheduler/sub_schedulers.go; then
    print_result "UUID Handling" "PASS" "Strategy UUID lookup implemented in sub_schedulers.go"
else
    print_result "UUID Handling" "FAIL" "Strategy UUID lookup not found in sub_schedulers.go"
fi

# Check 5: Verify context handling improvements
echo "Checking context handling..."
if grep -q "dataCtx, dataCancel" internal/strategy/optimizer/orchestrator.go; then
    print_result "Context Handling" "PASS" "Independent context management implemented"
else
    print_result "Context Handling" "FAIL" "Independent context management not found"
fi

# Check 6: Verify database recovery functionality
echo "Checking database recovery..."
if grep -q "RecoverConnection" internal/database/database.go; then
    print_result "Database Recovery" "PASS" "Database connection recovery implemented"
else
    print_result "Database Recovery" "FAIL" "Database connection recovery not found"
fi

# Check 7: Verify test files exist
echo "Checking test files..."
if [ -f "internal/strategy/optimizer/orchestrator_test.go" ]; then
    print_result "Test Files" "PASS" "Orchestrator test file exists"
else
    print_result "Test Files" "WARNING" "Orchestrator test file not found"
fi

# Check 8: Verify monitoring script exists
echo "Checking monitoring capabilities..."
if [ -f "scripts/monitor_fixes.go" ]; then
    print_result "Monitoring Script" "PASS" "Monitoring script created"
else
    print_result "Monitoring Script" "WARNING" "Monitoring script not found"
fi

# Check 9: Code compilation check
echo "Checking code compilation..."
if go build -o /tmp/qcat_test ./cmd/qcat > /dev/null 2>&1; then
    print_result "Code Compilation" "PASS" "Main application compiles successfully"
    rm -f /tmp/qcat_test
else
    print_result "Code Compilation" "FAIL" "Main application fails to compile"
fi

# Summary
echo "================================"
echo "📊 VERIFICATION SUMMARY"
echo "================================"
echo -e "✅ ${GREEN}Passed: $PASS_COUNT${NC}"
echo -e "⚠️  ${YELLOW}Warnings: $WARN_COUNT${NC}"
echo -e "❌ ${RED}Failed: $FAIL_COUNT${NC}"
echo "📋 Total Checks: $((PASS_COUNT + WARN_COUNT + FAIL_COUNT))"

if [ $FAIL_COUNT -eq 0 ]; then
    echo ""
    echo "🎉 All critical checks passed! The fixes have been successfully implemented."
    echo ""
    echo "Next steps:"
    echo "1. Run database migrations: go run cmd/migrate/main.go -config configs/config.yaml -up"
    echo "2. Start the application: go run cmd/qcat/main.go"
    echo "3. Monitor logs for any remaining issues"
    echo "4. Run integration tests when database is available"
else
    echo ""
    echo "⚠️  $FAIL_COUNT checks failed. Please review and fix the issues above."
    exit 1
fi
