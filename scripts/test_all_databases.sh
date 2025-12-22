#!/bin/bash
# Test All Databases Script
# Runs the complete test suite against SQLite, MySQL, and PostgreSQL
#
# Usage:
#   ./scripts/test_all_databases.sh              # Run all tests (assumes containers running)
#   ./scripts/test_all_databases.sh --sqlite     # SQLite only
#   ./scripts/test_all_databases.sh --mysql      # MySQL only
#   ./scripts/test_all_databases.sh --postgres   # PostgreSQL only
#   ./scripts/test_all_databases.sh --verbose    # Show all test output
#
# Prerequisites:
#   Docker containers must be running:
#     docker run -d --name mysql-test -p 3306:3306 -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=gassigeher_test -e MYSQL_USER=gassigeher -e MYSQL_PASSWORD=gassigeher mysql:8
#     docker run -d --name postgres-test -p 5432:5432 -e POSTGRES_DB=gassigeher_test -e POSTGRES_USER=gassigeher -e POSTGRES_PASSWORD=gassigeher postgres:15

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Default database connection settings (adjust ports if needed)
MYSQL_PORT=${MYSQL_PORT:-3307}
POSTGRES_PORT=${POSTGRES_PORT:-5433}
MYSQL_DSN="gassigeher:gassigeher@tcp(localhost:${MYSQL_PORT})/gassigeher_test?parseTime=true&multiStatements=true"
POSTGRES_DSN="postgres://gassigeher:gassigeher@localhost:${POSTGRES_PORT}/gassigeher_test?sslmode=disable"

# Parse arguments
RUN_SQLITE=true
RUN_MYSQL=true
RUN_POSTGRES=true
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --sqlite)
            RUN_SQLITE=true; RUN_MYSQL=false; RUN_POSTGRES=false; shift ;;
        --mysql)
            RUN_SQLITE=false; RUN_MYSQL=true; RUN_POSTGRES=false; shift ;;
        --postgres)
            RUN_SQLITE=false; RUN_MYSQL=false; RUN_POSTGRES=true; shift ;;
        --verbose|-v)
            VERBOSE=true; shift ;;
        --help|-h)
            echo "Usage: $0 [--sqlite|--mysql|--postgres] [--verbose]"
            echo ""
            echo "Options:"
            echo "  --sqlite    Run only SQLite tests"
            echo "  --mysql     Run only MySQL tests"
            echo "  --postgres  Run only PostgreSQL tests"
            echo "  --verbose   Show full test output"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

echo -e "${BOLD}${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BOLD}${CYAN}           Gassigeher Multi-Database Test Suite${NC}"
echo -e "${BOLD}${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Track results and timing
declare -A RESULTS
declare -A TIMES
declare -A COUNTS

run_tests() {
    local db_name=$1
    local env_var=$2
    local env_val=$3

    echo -e "${YELLOW}▶${NC} Testing with ${BOLD}$db_name${NC}..."

    local start_time=$(date +%s.%N)
    local output_file="/tmp/test_results_${db_name,,}.txt"

    if [ -n "$env_var" ]; then
        export "$env_var"="$env_val"
    fi

    if $VERBOSE; then
        if go test ./... -count=1 2>&1 | tee "$output_file"; then
            RESULTS[$db_name]="PASS"
        else
            RESULTS[$db_name]="FAIL"
        fi
    else
        if go test ./... -count=1 > "$output_file" 2>&1; then
            RESULTS[$db_name]="PASS"
        else
            RESULTS[$db_name]="FAIL"
        fi
    fi

    if [ -n "$env_var" ]; then
        unset "$env_var"
    fi

    local end_time=$(date +%s.%N)
    TIMES[$db_name]=$(echo "$end_time - $start_time" | bc | xargs printf "%.1f")

    # Count passed tests
    COUNTS[$db_name]=$(grep -c "^ok" "$output_file" 2>/dev/null || echo "0")

    if [ "${RESULTS[$db_name]}" = "PASS" ]; then
        echo -e "  ${GREEN}✓${NC} $db_name: ${GREEN}PASSED${NC} (${COUNTS[$db_name]} packages, ${TIMES[$db_name]}s)"
    else
        echo -e "  ${RED}✗${NC} $db_name: ${RED}FAILED${NC} (${TIMES[$db_name]}s)"
        if ! $VERBOSE; then
            echo -e "  ${CYAN}→${NC} See $output_file for details"
        fi
    fi
    echo ""
}

# Run tests
if $RUN_SQLITE; then
    run_tests "SQLite" "" ""
fi

if $RUN_MYSQL; then
    # Check if MySQL is available
    if nc -z localhost $MYSQL_PORT 2>/dev/null; then
        run_tests "MySQL" "DB_TEST_MYSQL" "$MYSQL_DSN"
    else
        echo -e "${YELLOW}▶${NC} Testing with ${BOLD}MySQL${NC}..."
        echo -e "  ${YELLOW}⚠${NC} MySQL: ${YELLOW}SKIPPED${NC} (port $MYSQL_PORT not available)"
        echo -e "  ${CYAN}→${NC} Start with: docker run -d --name mysql-test -p ${MYSQL_PORT}:3306 -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=gassigeher_test -e MYSQL_USER=gassigeher -e MYSQL_PASSWORD=gassigeher mysql:8"
        echo ""
        RESULTS["MySQL"]="SKIP"
    fi
fi

if $RUN_POSTGRES; then
    # Check if PostgreSQL is available
    if nc -z localhost $POSTGRES_PORT 2>/dev/null; then
        run_tests "PostgreSQL" "DB_TEST_POSTGRES" "$POSTGRES_DSN"
    else
        echo -e "${YELLOW}▶${NC} Testing with ${BOLD}PostgreSQL${NC}..."
        echo -e "  ${YELLOW}⚠${NC} PostgreSQL: ${YELLOW}SKIPPED${NC} (port $POSTGRES_PORT not available)"
        echo -e "  ${CYAN}→${NC} Start with: docker run -d --name postgres-test -p ${POSTGRES_PORT}:5432 -e POSTGRES_DB=gassigeher_test -e POSTGRES_USER=gassigeher -e POSTGRES_PASSWORD=gassigeher postgres:15"
        echo ""
        RESULTS["PostgreSQL"]="SKIP"
    fi
fi

# Summary
echo -e "${BOLD}${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BOLD}                         Summary${NC}"
echo -e "${BOLD}${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

PASSED=0
FAILED=0
SKIPPED=0
TOTAL_TIME=0

for db in SQLite MySQL PostgreSQL; do
    if [ -n "${RESULTS[$db]}" ]; then
        case "${RESULTS[$db]}" in
            PASS)
                echo -e "  ${GREEN}✓${NC} $db: ${GREEN}PASSED${NC} (${TIMES[$db]}s)"
                PASSED=$((PASSED + 1))
                TOTAL_TIME=$(echo "$TOTAL_TIME + ${TIMES[$db]}" | bc)
                ;;
            FAIL)
                echo -e "  ${RED}✗${NC} $db: ${RED}FAILED${NC} (${TIMES[$db]}s)"
                FAILED=$((FAILED + 1))
                TOTAL_TIME=$(echo "$TOTAL_TIME + ${TIMES[$db]}" | bc)
                ;;
            SKIP)
                echo -e "  ${YELLOW}○${NC} $db: ${YELLOW}SKIPPED${NC}"
                SKIPPED=$((SKIPPED + 1))
                ;;
        esac
    fi
done

echo ""
TOTAL_TIME_FMT=$(printf "%.1f" $TOTAL_TIME)
echo -e "  Total time: ${BOLD}${TOTAL_TIME_FMT}s${NC}"
echo ""

if [ $FAILED -gt 0 ]; then
    echo -e "${RED}${BOLD}Some tests failed!${NC}"
    exit 1
elif [ $PASSED -eq 0 ]; then
    echo -e "${YELLOW}${BOLD}No tests were run.${NC}"
    exit 1
else
    echo -e "${GREEN}${BOLD}All tests passed! ✅${NC}"
    exit 0
fi
