#!/bin/bash

################################################################################
#                                                                              #
#   ╔═══════════════════════════════════════════════════════════════════╗     #
#   ║                     GASSIGEHER TEST SUITE                         ║     #
#   ║                   Comprehensive Test Overview                     ║     #
#   ╚═══════════════════════════════════════════════════════════════════╝     #
#                                                                              #
################################################################################

set -euo pipefail

# Modern & Bold Color Scheme (Purple/Magenta theme)
PRIMARY='\033[1;35m'   # Bold Magenta - headers and emphasis
SECONDARY='\033[0;36m' # Cyan - secondary text
SUCCESS='\033[1;32m'   # Bold Green - success states
WARNING='\033[1;33m'   # Bold Yellow - warnings
ERROR='\033[1;31m'     # Bold Red - errors
INFO='\033[0;35m'      # Regular Magenta - info text
MUTED='\033[2;37m'     # Dim White - less important text
ACCENT='\033[1;36m'    # Bold Cyan - accents
WHITE='\033[1;37m'     # Bold White - important text
RESET='\033[0m'

# ASCII box drawing characters (compatible with all terminals)
BOX_H="="     # Horizontal line
BOX_V="|"     # Vertical line
BOX_TL="+"    # Top-left corner
BOX_TR="+"    # Top-right corner
BOX_BL="+"    # Bottom-left corner
BOX_BR="+"    # Bottom-right corner
BOX_VR="+"    # Vertical-right junction
BOX_VL="+"    # Vertical-left junction
BOX_HU="+"    # Horizontal-up junction
BOX_HD="+"    # Horizontal-down junction
BOX_CROSS="+" # Cross junction
THIN_H="-"    # Thin horizontal
THIN_V="|"    # Thin vertical

# Configuration
COVERAGE_WARN_THRESHOLD=50
COVERAGE_CRITICAL_THRESHOLD=30
RESULTS_DIR=".test-results"
PREVIOUS_RESULTS="$RESULTS_DIR/previous.json"
CURRENT_RESULTS="$RESULTS_DIR/current.json"
HTML_REPORT="$RESULTS_DIR/test-report.html"

# Global variables
declare -A PACKAGE_RESULTS
declare -A PACKAGE_COVERAGE
declare -A PACKAGE_TIMES
declare -A FAILED_TESTS
TOTAL_TESTS=0
TOTAL_PASSED=0
TOTAL_FAILED=0
START_TIME=$(date +%s)

################################################################################
# Helper Functions
################################################################################

print_header() {
  local title="$1"
  local inner_width=76 # Total width minus 2 border chars
  local title_len=${#title}
  local padding_left=$(((inner_width - title_len) / 2))
  local padding_right=$((inner_width - title_len - padding_left))

  echo ""
  echo -e "${PRIMARY}${BOX_TL}$(printf "%-${inner_width}s" | tr ' ' "$BOX_H")${BOX_TR}${RESET}"
  printf "${PRIMARY}${BOX_V}${RESET}%${padding_left}s${WHITE}%s${RESET}%${padding_right}s${PRIMARY}${BOX_V}${RESET}\n" "" "$title" ""
  echo -e "${PRIMARY}${BOX_BL}$(printf "%-${inner_width}s" | tr ' ' "$BOX_H")${BOX_BR}${RESET}"
  echo ""
}

print_separator() {
  echo -e "${MUTED}$(printf '%78s' | tr ' ' "$THIN_H")${RESET}"
}

print_double_separator() {
  echo -e "${PRIMARY}$(printf '%78s' | tr ' ' "$BOX_H")${RESET}"
}

print_progress_bar() {
  local current=$1
  local total=$2
  local width=50
  local percentage=$((current * 100 / total))
  local filled=$((width * current / total))
  local empty=$((width - filled))

  printf "\r  ${ACCENT}[${RESET}"
  printf "%${filled}s" | tr ' ' '#'
  printf "%${empty}s" | tr ' ' '.'
  printf "${ACCENT}]${RESET} ${WHITE}%3d%%${RESET} ${MUTED}[%d/%d]${RESET}  " "$percentage" "$current" "$total"
}

spinner() {
  local pid=$1
  local delay=0.1
  local spinstr='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
  while ps -p $pid >/dev/null 2>&1; do
    local temp=${spinstr#?}
    printf " ${CYAN}%c${RESET} " "$spinstr"
    spinstr=$temp${spinstr%"$temp"}
    sleep $delay
    printf "\b\b\b\b"
  done
  printf "    \b\b\b\b"
}

format_duration() {
  local seconds=$1
  if [ $seconds -lt 1 ]; then
    echo "${seconds}ms"
  elif [ $seconds -lt 60 ]; then
    printf "%.2fs" "$seconds"
  else
    local mins=$((seconds / 60))
    local secs=$((seconds % 60))
    printf "%dm %ds" "$mins" "$secs"
  fi
}

get_coverage_color() {
  local coverage=$1
  # Default to 0 if empty
  [ -z "$coverage" ] && coverage=0
  # Use awk for float comparison (more portable than bc)
  if awk "BEGIN {exit !($coverage >= $COVERAGE_WARN_THRESHOLD)}"; then
    echo "$SUCCESS"
  elif awk "BEGIN {exit !($coverage >= $COVERAGE_CRITICAL_THRESHOLD)}"; then
    echo "$WARNING"
  else
    echo "$ERROR"
  fi
}

################################################################################
# Database Detection
################################################################################

detect_databases() {
  local databases=("sqlite")

  echo -e "${INFO}▸ Detecting available databases...${RESET}" >&2

  # Check for MySQL
  if [ -n "${DB_TEST_MYSQL:-}" ]; then
    if timeout 2 bash -c "echo > /dev/tcp/localhost/3306" 2>/dev/null; then
      databases+=("mysql")
      echo -e "  ${SUCCESS}✓${RESET} MySQL detected" >&2
    else
      echo -e "  ${WARNING}⚠${RESET} MySQL configured but not available" >&2
    fi
  fi

  # Check for PostgreSQL
  if [ -n "${DB_TEST_POSTGRES:-}" ]; then
    if timeout 2 bash -c "echo > /dev/tcp/localhost/5432" 2>/dev/null; then
      databases+=("postgres")
      echo -e "  ${SUCCESS}✓${RESET} PostgreSQL detected" >&2
    else
      echo -e "  ${WARNING}⚠${RESET} PostgreSQL configured but not available" >&2
    fi
  fi

  echo -e "  ${ACCENT}▸${RESET} Testing with: ${WHITE}${databases[*]}${RESET}\n" >&2
  echo "${databases[@]}"
}

################################################################################
# Test Execution
################################################################################

run_package_tests() {
  local package=$1
  local db_type=$2
  local output_file=$(mktemp)
  local coverage_file=$(mktemp)
  local start_time=$(date +%s)

  # Set database environment for test
  export DB_TYPE="$db_type"

  # Run tests with coverage
  if go test -v -coverprofile="$coverage_file" "$package" >"$output_file" 2>&1; then
    local status="PASS"
  else
    local status="FAIL"
  fi

  local end_time=$(date +%s)
  local duration=$((end_time - start_time))

  # Parse results - count top-level test results only (not subtests)
  local passed=$(grep -c "^--- PASS:" "$output_file" 2>/dev/null || echo "0")
  local failed=$(grep -c "^--- FAIL:" "$output_file" 2>/dev/null || echo "0")
  # Remove any whitespace/newlines
  passed=$(echo "$passed" | tr -d '[:space:]')
  failed=$(echo "$failed" | tr -d '[:space:]')
  # Default to 0 if empty
  [ -z "$passed" ] && passed=0
  [ -z "$failed" ] && failed=0

  # Calculate coverage
  local coverage="0"
  if [ -f "$coverage_file" ] && [ -s "$coverage_file" ]; then
    coverage=$(go tool cover -func="$coverage_file" 2>/dev/null | grep "total:" | awk '{print $NF}' | sed 's/%//' || echo "0")
    # Ensure coverage is not empty
    [ -z "$coverage" ] && coverage="0"
  fi

  # Store results
  local key="${package}_${db_type}"
  PACKAGE_RESULTS[$key]="$status|$passed|$failed"
  PACKAGE_COVERAGE[$key]="$coverage"
  PACKAGE_TIMES[$key]="$duration"

  # Extract failed test names
  if [ "$status" == "FAIL" ]; then
    local failed_names=$(grep "^--- FAIL:" "$output_file" | awk '{print $3}' | tr '\n' ',' | sed 's/,$//')
    FAILED_TESTS[$key]="$failed_names"
  fi

  # Update totals
  TOTAL_TESTS=$((TOTAL_TESTS + passed + failed))
  TOTAL_PASSED=$((TOTAL_PASSED + passed))
  TOTAL_FAILED=$((TOTAL_FAILED + failed))

  # Cleanup
  rm -f "$output_file" "$coverage_file"

  # Return status via global variable instead of echo (to avoid subshell)
  LAST_TEST_STATUS="$status"
}

################################################################################
# Main Test Runner
################################################################################

run_all_tests() {
  print_header "TEST EXECUTION IN PROGRESS"

  # Detect databases
  local databases=($(detect_databases))

  # Get all test packages
  local packages=($(go list ./... 2>/dev/null | grep -v vendor || echo ""))
  local total_runs=$((${#packages[@]} * ${#databases[@]}))
  local current_run=0

  if [ ${#packages[@]} -eq 0 ]; then
    echo -e "${ERROR}✗ No test packages found${RESET}"
    return 1
  fi

  echo -e "${INFO}▸ Found ${WHITE}${#packages[@]}${RESET} packages"
  echo -e "${INFO}▸ Testing with ${WHITE}${#databases[@]}${RESET} database(s)"
  echo ""
  print_double_separator

  # Run tests for each package (grouped by package)
  for package in "${packages[@]}"; do
    local pkg_name=$(echo "$package" | sed "s|^.*/gassigeher/||")
    local pkg_tests=0
    local pkg_passed=0
    local pkg_failed=0

    # Package header
    echo ""
    echo -e "${PRIMARY}▸ ${WHITE}${pkg_name}${RESET}"

    for db_type in "${databases[@]}"; do
      current_run=$((current_run + 1))

      # Show database being tested
      echo -ne "  ${MUTED}${db_type}${RESET} "

      # Run tests (modifies global variables directly, no subshell)
      run_package_tests "$package" "$db_type"
      local status="$LAST_TEST_STATUS"

      # Get test count for this run
      local key="${package}_${db_type}"
      local result="${PACKAGE_RESULTS[$key]}"
      IFS='|' read -r status passed failed <<<"$result"

      pkg_tests=$((pkg_tests + passed + failed))
      pkg_passed=$((pkg_passed + passed))
      pkg_failed=$((pkg_failed + failed))

      # Show result inline
      if [ "$status" == "PASS" ]; then
        if [ "$passed" -gt 0 ]; then
          echo -e "${SUCCESS}✓${RESET} ${WHITE}${passed}${RESET} tests passed"
        else
          echo -e "${MUTED}◦ no tests${RESET}"
        fi
      else
        echo -e "${ERROR}✗${RESET} ${ERROR}${failed}${RESET} tests failed"
      fi
    done

    # Package summary if multiple databases
    if [ ${#databases[@]} -gt 1 ] && [ $pkg_tests -gt 0 ]; then
      echo -e "  ${MUTED}└-${RESET} Total: ${WHITE}${pkg_passed}${RESET}/${WHITE}${pkg_tests}${RESET} passed"
    fi
  done

  echo ""
  print_double_separator
  print_progress_bar "$total_runs" "$total_runs"
  echo -e "\n"
  echo -e "${SUCCESS}✓ Test execution completed${RESET}"
  echo ""
}

################################################################################
# Results Summary
################################################################################

print_summary() {
  print_header "TEST RESULTS SUMMARY"

  local total_time=$(($(date +%s) - START_TIME))
  local pass_rate=0
  if [ $TOTAL_TESTS -gt 0 ]; then
    pass_rate=$((TOTAL_PASSED * 100 / TOTAL_TESTS))
  fi

  # Modern statistics cards
  echo -e "${PRIMARY}+==================+==================+==================+==================+${RESET}"
  printf "${PRIMARY}|${RESET}%-18s${PRIMARY}|${RESET}%-18s${PRIMARY}|${RESET}%-18s${PRIMARY}|${RESET}%-18s${PRIMARY}|${RESET}\n" \
    "  TOTAL TESTS    " "  PASSED         " "  FAILED         " "  PASS RATE      "
  echo -e "${PRIMARY}+==================+==================+==================+==================+${RESET}"

  # Build the value row
  local col1=$(printf "     %-10d  " "$TOTAL_TESTS")
  local col2=$(printf "     %-10d  " "$TOTAL_PASSED")

  if [ $TOTAL_FAILED -eq 0 ]; then
    local col3=$(printf "     %-10d  " "$TOTAL_FAILED")
  else
    local col3=$(printf "     %-10d  " "$TOTAL_FAILED")
  fi

  local col4
  if [ $pass_rate -ge 90 ]; then
    col4=$(printf "      %3d%%       " "$pass_rate")
  elif [ $pass_rate -ge 70 ]; then
    col4=$(printf "      %3d%%       " "$pass_rate")
  else
    col4=$(printf "      %3d%%       " "$pass_rate")
  fi

  printf "${PRIMARY}|${RESET}%-18s${PRIMARY}|${RESET}%-18s${PRIMARY}|${RESET}%-18s${PRIMARY}|${RESET}%-18s${PRIMARY}|${RESET}\n" \
    "$col1" "$col2" "$col3" "$col4"

  echo -e "${PRIMARY}+==================+==================+==================+==================+${RESET}"

  echo ""
  echo -e "  ${MUTED}Duration:${RESET} ${WHITE}$(format_duration $total_time)${RESET}"
  echo ""
}

print_detailed_results() {
  print_header "DETAILED PACKAGE RESULTS"

  # Table header with fixed column widths
  echo -e "${PRIMARY}+=================================+==========+==========+=========+========+${RESET}"
  printf "${PRIMARY}|${RESET}%-34s${PRIMARY}|${RESET}%-10s${PRIMARY}|${RESET}%-10s${PRIMARY}|${RESET}%-9s${PRIMARY}|${RESET}%-8s${PRIMARY}|${RESET}\n" \
    " Package                        " " Database" " Coverage" " Tests  " " Time  "
  echo -e "${PRIMARY}+=================================+==========+==========+=========+========+${RESET}"

  # Sort packages
  local sorted_keys=($(printf '%s\n' "${!PACKAGE_RESULTS[@]}" | sort))

  for key in "${sorted_keys[@]}"; do
    local package=$(echo "$key" | sed 's/_sqlite$//' | sed 's/_mysql$//' | sed 's/_postgres$//')
    local db_type=$(echo "$key" | grep -o '[^_]*$')
    local result="${PACKAGE_RESULTS[$key]}"
    local coverage="${PACKAGE_COVERAGE[$key]}"
    local duration="${PACKAGE_TIMES[$key]}"

    IFS='|' read -r status passed failed <<<"$result"

    # Format package name (truncate if too long)
    local pkg_display="$package"
    if [ ${#pkg_display} -gt 31 ]; then
      pkg_display="${pkg_display:0:28}..."
    fi

    # Status indicator
    local status_icon
    if [ "$status" == "PASS" ]; then
      status_icon="✓"
    else
      status_icon="✗"
    fi

    # Coverage color indicator
    local cov_str=$(printf "%6.1f%%" "$coverage")

    # Format duration
    local time_str=$(format_duration "$duration")

    # Build the row with fixed widths
    local col1=$(printf " %s %-30s" "$status_icon" "$pkg_display")
    local col2=$(printf " %-8s" "$db_type")
    local col3=$(printf " %8s" "$cov_str")
    local col4=$(printf " %3d/%-3d" "$passed" "$((passed + failed))")
    local col5=$(printf " %-6s" "$time_str")

    printf "${PRIMARY}|${RESET}%-34s${PRIMARY}|${RESET}%-10s${PRIMARY}|${RESET}%-10s${PRIMARY}|${RESET}%-9s${PRIMARY}|${RESET}%-8s${PRIMARY}|${RESET}\n" \
      "$col1" "$col2" "$col3" "$col4" "$col5"

    # Show failed tests if any
    if [ "$status" == "FAIL" ] && [ -n "${FAILED_TESTS[$key]:-}" ]; then
      IFS=',' read -ra fails <<<"${FAILED_TESTS[$key]}"
      for test_name in "${fails[@]}"; do
        local fail_line=$(printf " ▸ Failed: %-60s" "$test_name")
        printf "${PRIMARY}|${RESET}%-72s${PRIMARY}|${RESET}\n" "$fail_line"
      done
    fi
  done

  echo -e "${PRIMARY}+=================================+==========+==========+=========+========+${RESET}\n"
}

print_coverage_warnings() {
  local has_warnings=0

  for key in "${!PACKAGE_COVERAGE[@]}"; do
    local coverage="${PACKAGE_COVERAGE[$key]}"
    [ -z "$coverage" ] && coverage=0
    if awk "BEGIN {exit !($coverage < $COVERAGE_WARN_THRESHOLD)}"; then
      has_warnings=1
      break
    fi
  done

  if [ $has_warnings -eq 0 ]; then
    return
  fi

  print_header "COVERAGE WARNINGS"

  local inner_width=76
  echo -e "${WARNING}+$(printf "%-${inner_width}s" | tr ' ' '=')+${RESET}"
  local header_text="  Packages with coverage below ${COVERAGE_WARN_THRESHOLD}%"
  printf "${WARNING}|${RESET}%-${inner_width}s${WARNING}|${RESET}\n" "$header_text"
  echo -e "${WARNING}+$(printf "%-${inner_width}s" | tr ' ' '=')+${RESET}"

  for key in "${!PACKAGE_COVERAGE[@]}"; do
    local coverage="${PACKAGE_COVERAGE[$key]}"
    [ -z "$coverage" ] && coverage=0
    if awk "BEGIN {exit !($coverage < $COVERAGE_WARN_THRESHOLD)}"; then
      local package=$(echo "$key" | sed 's/_[^_]*$//')
      local line=$(printf "  ⚠  %-60s %6.1f%%" "$package" "$coverage")
      printf "${WARNING}|${RESET}%-${inner_width}s${WARNING}|${RESET}\n" "$line"
    fi
  done

  echo -e "${WARNING}+$(printf "%-${inner_width}s" | tr ' ' '=')+${RESET}\n"
}

################################################################################
# Comparison with Previous Run
################################################################################

save_current_results() {
  mkdir -p "$RESULTS_DIR"

  # Save previous results if current exists
  if [ -f "$CURRENT_RESULTS" ]; then
    cp "$CURRENT_RESULTS" "$PREVIOUS_RESULTS"
  fi

  # Save current results
  cat >"$CURRENT_RESULTS" <<EOF
{
    "timestamp": "$(date -Iseconds)",
    "total_tests": $TOTAL_TESTS,
    "total_passed": $TOTAL_PASSED,
    "total_failed": $TOTAL_FAILED,
    "duration": $(($(date +%s) - START_TIME)),
    "packages": {
EOF

  local first=1
  for key in "${!PACKAGE_RESULTS[@]}"; do
    [ $first -eq 0 ] && echo "," >>"$CURRENT_RESULTS"
    first=0

    local result="${PACKAGE_RESULTS[$key]}"
    local coverage="${PACKAGE_COVERAGE[$key]}"
    local duration="${PACKAGE_TIMES[$key]}"

    IFS='|' read -r status passed failed <<<"$result"

    cat >>"$CURRENT_RESULTS" <<EOF
        "$key": {
            "status": "$status",
            "passed": $passed,
            "failed": $failed,
            "coverage": $coverage,
            "duration": $duration
        }
EOF
  done

  cat >>"$CURRENT_RESULTS" <<EOF

    }
}
EOF
}

print_comparison() {
  if [ ! -f "$PREVIOUS_RESULTS" ]; then
    return
  fi

  print_header "COMPARISON WITH PREVIOUS RUN"

  local prev_total=$(grep '"total_tests"' "$PREVIOUS_RESULTS" | grep -o '[0-9]*')
  local prev_passed=$(grep '"total_passed"' "$PREVIOUS_RESULTS" | grep -o '[0-9]*')
  local prev_failed=$(grep '"total_failed"' "$PREVIOUS_RESULTS" | grep -o '[0-9]*')

  echo -e "${PRIMARY}+====================+===============+===============+===============+${RESET}"
  printf "${PRIMARY}|${RESET}%-20s${PRIMARY}|${RESET}%-15s${PRIMARY}|${RESET}%-15s${PRIMARY}|${RESET}%-15s${PRIMARY}|${RESET}\n" \
    " Metric            " " Previous     " " Current      " " Change       "
  echo -e "${PRIMARY}+====================+===============+===============+===============+${RESET}"

  # Total tests comparison
  local diff=$((TOTAL_TESTS - prev_total))
  local change_str=$([ $diff -ge 0 ] && echo "+$diff" || echo "$diff")
  local row1=$(printf " %-18s" "Total Tests")
  local row2=$(printf " %12d " "$prev_total")
  local row3=$(printf " %12d " "$TOTAL_TESTS")
  local row4=$(printf " %12s " "$change_str")
  printf "${PRIMARY}|${RESET}%-20s${PRIMARY}|${RESET}%-15s${PRIMARY}|${RESET}%-15s${PRIMARY}|${RESET}%-15s${PRIMARY}|${RESET}\n" \
    "$row1" "$row2" "$row3" "$row4"

  # Passed tests comparison
  diff=$((TOTAL_PASSED - prev_passed))
  change_str=$([ $diff -ge 0 ] && echo "+$diff" || echo "$diff")
  row1=$(printf " %-18s" "Passed")
  row2=$(printf " %12d " "$prev_passed")
  row3=$(printf " %12d " "$TOTAL_PASSED")
  row4=$(printf " %12s " "$change_str")
  printf "${PRIMARY}|${RESET}%-20s${PRIMARY}|${RESET}%-15s${PRIMARY}|${RESET}%-15s${PRIMARY}|${RESET}%-15s${PRIMARY}|${RESET}\n" \
    "$row1" "$row2" "$row3" "$row4"

  # Failed tests comparison
  diff=$((TOTAL_FAILED - prev_failed))
  change_str=$([ $diff -ge 0 ] && echo "+$diff" || echo "$diff")
  row1=$(printf " %-18s" "Failed")
  row2=$(printf " %12d " "$prev_failed")
  row3=$(printf " %12d " "$TOTAL_FAILED")
  row4=$(printf " %12s " "$change_str")
  printf "${PRIMARY}|${RESET}%-20s${PRIMARY}|${RESET}%-15s${PRIMARY}|${RESET}%-15s${PRIMARY}|${RESET}%-15s${PRIMARY}|${RESET}\n" \
    "$row1" "$row2" "$row3" "$row4"

  echo -e "${PRIMARY}+====================+===============+===============+===============+${RESET}\n"
}

################################################################################
# HTML Report Generation
################################################################################

generate_html_report() {
  mkdir -p "$RESULTS_DIR"

  cat >"$HTML_REPORT" <<'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Gassigeher Test Report</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            padding: 20px;
            color: #333;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        .header h1 { font-size: 2.5em; margin-bottom: 10px; }
        .header p { font-size: 1.1em; opacity: 0.9; }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            padding: 30px;
            background: #f8f9fa;
        }
        .stat-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            text-align: center;
        }
        .stat-card h3 { color: #666; font-size: 0.9em; margin-bottom: 10px; }
        .stat-card .value { font-size: 2em; font-weight: bold; }
        .stat-card.pass .value { color: #28a745; }
        .stat-card.fail .value { color: #dc3545; }
        .stat-card.time .value { color: #667eea; }
        .table-container { padding: 30px; }
        table {
            width: 100%;
            border-collapse: collapse;
            background: white;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            border-radius: 8px;
            overflow: hidden;
        }
        thead {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        th, td {
            padding: 15px;
            text-align: left;
        }
        tbody tr:nth-child(even) { background: #f8f9fa; }
        tbody tr:hover { background: #e9ecef; }
        .status-pass { color: #28a745; font-weight: bold; }
        .status-fail { color: #dc3545; font-weight: bold; }
        .coverage-high { color: #28a745; font-weight: bold; }
        .coverage-medium { color: #ffc107; font-weight: bold; }
        .coverage-low { color: #dc3545; font-weight: bold; }
        .footer {
            background: #f8f9fa;
            padding: 20px;
            text-align: center;
            color: #666;
            border-top: 1px solid #dee2e6;
        }
        .failed-tests {
            margin-top: 5px;
            padding: 5px 10px;
            background: #fff3cd;
            border-left: 3px solid #ffc107;
            font-size: 0.85em;
            color: #856404;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🐕 Gassigeher Test Report</h1>
            <p>Generated on TIMESTAMP</p>
        </div>

        <div class="stats">
            <div class="stat-card">
                <h3>Total Tests</h3>
                <div class="value">TOTAL_TESTS</div>
            </div>
            <div class="stat-card pass">
                <h3>Passed</h3>
                <div class="value">TOTAL_PASSED</div>
            </div>
            <div class="stat-card fail">
                <h3>Failed</h3>
                <div class="value">TOTAL_FAILED</div>
            </div>
            <div class="stat-card time">
                <h3>Duration</h3>
                <div class="value">DURATION</div>
            </div>
        </div>

        <div class="table-container">
            <table>
                <thead>
                    <tr>
                        <th>Package</th>
                        <th>Database</th>
                        <th>Status</th>
                        <th>Coverage</th>
                        <th>Tests</th>
                        <th>Duration</th>
                    </tr>
                </thead>
                <tbody>
                    TABLE_ROWS
                </tbody>
            </table>
        </div>

        <div class="footer">
            <p>Report generated by Gassigeher Test Suite</p>
        </div>
    </div>
</body>
</html>
EOF

  # Replace placeholders
  sed -i "s/TIMESTAMP/$(date '+%Y-%m-%d %H:%M:%S')/" "$HTML_REPORT"
  sed -i "s/TOTAL_TESTS/$TOTAL_TESTS/" "$HTML_REPORT"
  sed -i "s/TOTAL_PASSED/$TOTAL_PASSED/" "$HTML_REPORT"
  sed -i "s/TOTAL_FAILED/$TOTAL_FAILED/" "$HTML_REPORT"
  sed -i "s/DURATION/$(format_duration $(($(date +%s) - START_TIME)))/" "$HTML_REPORT"

  # Generate table rows
  local rows=""
  for key in $(printf '%s\n' "${!PACKAGE_RESULTS[@]}" | sort); do
    local package=$(echo "$key" | sed 's/_[^_]*$//')
    local db_type=$(echo "$key" | grep -o '[^_]*$')
    local result="${PACKAGE_RESULTS[$key]}"
    local coverage="${PACKAGE_COVERAGE[$key]}"
    local duration="${PACKAGE_TIMES[$key]}"

    IFS='|' read -r status passed failed <<<"$result"

    local status_class=$([ "$status" == "PASS" ] && echo "status-pass" || echo "status-fail")
    [ -z "$coverage" ] && coverage=0
    local cov_class="coverage-high"
    if awk "BEGIN {exit !($coverage < $COVERAGE_WARN_THRESHOLD)}"; then
      cov_class="coverage-medium"
    fi
    if awk "BEGIN {exit !($coverage < $COVERAGE_CRITICAL_THRESHOLD)}"; then
      cov_class="coverage-low"
    fi

    rows+="<tr>"
    rows+="<td>$package</td>"
    rows+="<td>$db_type</td>"
    rows+="<td class=\"$status_class\">$status</td>"
    rows+="<td class=\"$cov_class\">$(printf "%.1f%%" "$coverage")</td>"
    rows+="<td>$passed/$((passed + failed))</td>"
    rows+="<td>$(format_duration "$duration")</td>"
    rows+="</tr>"

    # Add failed tests row if any
    if [ "$status" == "FAIL" ] && [ -n "${FAILED_TESTS[$key]:-}" ]; then
      rows+="<tr><td colspan=\"6\" class=\"failed-tests\">"
      rows+="<strong>Failed tests:</strong> ${FAILED_TESTS[$key]//,/, }"
      rows+="</td></tr>"
    fi
  done

  sed -i "s|TABLE_ROWS|$rows|" "$HTML_REPORT"

  echo -e "${SUCCESS}✓${RESET} HTML report generated: ${ACCENT}$HTML_REPORT${RESET}"
}

################################################################################
# Main Execution
################################################################################

main() {
  clear

  # Modern ASCII Art Header
  echo -e "${PRIMARY}"
  cat <<"EOF"
    +===========================================================================+
    |                                                                           |
    |     ██████╗  █████╗ ███████╗███████╗██╗ ██████╗ ███████╗██╗  ██╗          |
    |    ██╔════╝ ██╔══██╗██╔════╝██╔════╝██║██╔════╝ ██╔════╝██║  ██║          |
    |    ██║  ███╗███████║███████╗███████╗██║██║  ███╗█████╗  ███████║          |
    |    ██║   ██║██╔══██║╚════██║╚════██║██║██║   ██║██╔══╝  ██╔══██║          |
    |    ╚██████╔╝██║  ██║███████║███████║██║╚██████╔╝███████╗██║  ██║          |
    |     ╚═════╝ ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝          |
    |                                                                           |
EOF
  echo -e "    |${RESET}${WHITE}                       COMPREHENSIVE TEST SUITE${RESET}${PRIMARY}                            |"
  echo -e "    |                                                                           |"
  echo -e "    +===========================================================================+${RESET}"
  echo ""

  # Run tests
  run_all_tests

  # Print results
  print_summary
  print_detailed_results
  print_coverage_warnings
  print_comparison

  # Save results
  save_current_results

  # Generate HTML report
  echo ""
  generate_html_report

  # Final summary
  echo ""
  print_double_separator
  if [ $TOTAL_FAILED -eq 0 ]; then
    echo -e "${SUCCESS}"
    echo "    ✓  ALL ${TOTAL_TESTS} TESTS PASSED!  ✓"
    echo -e "${RESET}"
  else
    echo -e "${ERROR}"
    echo "    ✗  ${TOTAL_FAILED} of ${TOTAL_TESTS} TESTS FAILED  ✗"
    echo -e "${RESET}"
  fi
  print_double_separator
  echo ""

  # Exit with appropriate code
  [ $TOTAL_FAILED -eq 0 ] && exit 0 || exit 1
}

# Run main function
main
