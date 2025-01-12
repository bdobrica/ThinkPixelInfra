#!/usr/bin/env bash
set -euo pipefail

#####################################
# Environment variable defaults
#####################################
MYSQL_HOST="${MYSQL_HOST:-localhost}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-root}"
MYSQL_DATABASE="${MYSQL_DATABASE:-thinkpixel}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
SQL_INIT_FILE="${SCRIPT_DIR}/db_thinkpixel.sql"
PATCHES_DIR="${SCRIPT_DIR}/patches"

#####################################
# Helper functions
#####################################

# Print text in red color.
function echo_red() {
  echo -e "\033[0;31m$*\033[0m"
}

# Print text in yellow color.
function echo_yellow() {
  echo -e "\033[0;33m$*\033[0m"
}

# Print text in green color.
function echo_green() {
  echo -e "\033[0;32m$*\033[0m"
}

# Run a simple MySQL query and print results (suppressing warnings).
function mysql_query() {
  # Usage: mysql_query "SQL_STATEMENT"
  mysql \
    --host="$MYSQL_HOST" \
    --port="$MYSQL_PORT" \
    --user="$MYSQL_USER" \
    --password="$MYSQL_PASSWORD" \
    --skip-column-names \
    --batch \
    -e "$1"
}

# Check if DB exists by looking it up in INFORMATION_SCHEMA
function db_exists() {
  local result
  result="$(mysql_query "SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = '${MYSQL_DATABASE}'")"
  if [[ -n "$result" ]]; then
    return 0
  else
    return 1
  fi
}

# Fetch the current DB version from wp_thinkpixel_db_versions, picking the row
# with the latest `updated_at`. Returns empty string if none found.
function get_current_version() {
  local sql="
    SELECT version
      FROM \`${MYSQL_DATABASE}\`.\`wp_thinkpixel_db_versions\`
     ORDER BY updated_at DESC
     LIMIT 1;
  "
  local ver
  ver="$(mysql_query "$sql" 2>/dev/null || true)"
  echo "$ver"
}

# Insert a new row into wp_thinkpixel_db_versions (if it doesn’t exist)
# or update the existing row to the given version.
function update_db_version() {
  local new_version="$1"
  # We'll just do an insert. If your schema only ever holds a single row,
  # you might do an UPSERT or check existence first. Adjust as needed.
  local sql="
    INSERT INTO \`${MYSQL_DATABASE}\`.\`wp_thinkpixel_db_versions\` (version, created_at, updated_at)
    VALUES ('${new_version}', NOW(), NOW())
    ON DUPLICATE KEY UPDATE
      version = '${new_version}',
      updated_at = NOW();
  "
  mysql_query "$sql"
}

#####################################
# Main logic
#####################################

echo_green "=== [Init Script] Checking database '${MYSQL_DATABASE}' existence ==="
if ! db_exists; then
  echo_yellow "Database '${MYSQL_DATABASE}' does NOT exist; creating and initializing..."
  # Create DB + run init script
  # If your init script itself does CREATE DATABASE, you can skip the CREATE DB line here.
  mysql_query "CREATE DATABASE IF NOT EXISTS \`${MYSQL_DATABASE}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
  mysql \
    --host="$MYSQL_HOST" \
    --port="$MYSQL_PORT" \
    --user="$MYSQL_USER" \
    --password="$MYSQL_PASSWORD" \
    "$MYSQL_DATABASE" < "${SQL_INIT_FILE}" || {
        echo_red "Failed to initialize database '${MYSQL_DATABASE}' from '${SQL_INIT_FILE}'"
        exit 1
    }
  echo_green "Database '${MYSQL_DATABASE}' created and initialized."
else
  echo_green "Database '${MYSQL_DATABASE}' already exists."
fi

echo_green "=== [Init Script] Checking current DB version ==="
current_version="$(get_current_version)"
if [[ -z "$current_version" ]]; then
  echo_yellow "No version found in 'wp_thinkpixel_db_versions'; defaulting to 0.0.0"
  current_version="0.0.0"
fi
echo "Current DB version is: ${current_version}"

# Gather all .sql patches that have a line like:
#   -- Version: 1.2.3
shopt -s nullglob
patch_files=( "${PATCHES_DIR}"/*.sql )

# We'll create a temporary list of "version|filepath" lines, filter them by version > current_version, then sort by version.
declare -a patch_list=()

for patch_file in "${patch_files[@]}"; do
  # Extract the line `-- Version: X.Y.Z`
  patch_ver_line="$(grep -E '^-- Version:' "${patch_file}" || true)"
  if [[ -n "$patch_ver_line" ]]; then
    # Extract "X.Y.Z" from that line.
    # e.g. "-- Version: 1.2.3" -> "1.2.3"
    patch_ver="${patch_ver_line#-- Version: }"

    # Compare with current_version. If patch_ver > current_version, queue it.
    # We’ll do the final filtering after we gather them, for convenience.
    patch_list+=( "${patch_ver}|${patch_file}" )
  fi
done

# Use 'sort -V' to sort by version in ascending order. Then we’ll apply only those strictly greater than $current_version.
if [[ ${#patch_list[@]} -eq 0 ]]; then
  echo "No patch files found in '${PATCHES_DIR}' or no valid '-- Version:' lines found."
  exit 0
else
  echo "=== [Init Script] Found ${#patch_list[@]} patch files with version tags ==="
  # Print them for debug
  for line in "${patch_list[@]}"; do
    echo "  $line"
  done

  # Sort them by version (the part before the '|').
  # Transform array into lines, sort, transform back into array.
  sorted_patches="$(printf '%s\n' "${patch_list[@]}" | sort -t'|' -k1 -V)"

  # Now iterate through them in sorted (ascending) order
  while IFS= read -r line; do
    patch_ver="${line%%|*}"
    patch_path="${line#*|}"

    # Compare patch_ver to current_version:
    # If patch_ver <= current_version, skip it.
    # We'll rely on "sort -V" plus a simple check below.
    # We can do a naive "dpkg --compare-versions" or some custom function,
    # but here's a quick numeric compare approach:
    if printf '%s\n%s\n' "$current_version" "$patch_ver" | sort -C -V; then
      # Means current_version <= patch_ver
      if [[ "$patch_ver" == "$current_version" ]]; then
        # Exactly equal, skip
        echo "[SKIP] Patch '${patch_path}' has version == current (${patch_ver})"
      else
        # current_version < patch_ver, apply patch
        echo "[APPLY] Patch '${patch_path}' (version ${patch_ver})"
        mysql \
          --host="$MYSQL_HOST" \
          --port="$MYSQL_PORT" \
          --user="$MYSQL_USER" \
          --password="$MYSQL_PASSWORD" \
          "$MYSQL_DATABASE" < "$patch_path" || {
              echo_red "Failed to apply patch '${patch_path}'"
              exit 1
          }
        echo_green "  - Patch applied. Updating DB version to '${patch_ver}'..."
        update_db_version "$patch_ver"
        current_version="$patch_ver"
      fi
    else
      # Means patch_ver < current_version
      echo "[SKIP] Patch '${patch_path}' has version < current (${patch_ver} < ${current_version})"
    fi
  done <<< "$sorted_patches"
fi

echo_green "=== [Init Script] Database migrations complete. Current version is '${current_version}'. ==="
