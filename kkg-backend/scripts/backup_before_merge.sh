#!/usr/bin/env sh
set -eu

backup_dir="${BACKUP_DIR:-./backups}"
timestamp="$(date +%Y%m%d_%H%M%S)"
db_host="${MYSQL_HOST:-127.0.0.1}"
db_port="${MYSQL_PORT:-3307}"
db_name="${MYSQL_DATABASE:-blog}"
db_user="${MYSQL_USER:-root}"
db_password="${MYSQL_PASSWORD:-root123456}"

mkdir -p "$backup_dir"

out="$backup_dir/${db_name}_before_merge_${timestamp}.sql"

mysqldump \
  --host="$db_host" \
  --port="$db_port" \
  --user="$db_user" \
  --password="$db_password" \
  --single-transaction \
  --routines \
  --triggers \
  --events \
  "$db_name" > "$out"

echo "backup written: $out"

