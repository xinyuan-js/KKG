#!/usr/bin/env sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 <backup.sql>" >&2
  exit 1
fi

backup_file="$1"
db_host="${MYSQL_HOST:-127.0.0.1}"
db_port="${MYSQL_PORT:-3307}"
db_name="${MYSQL_DATABASE:-blog}"
db_user="${MYSQL_USER:-root}"
db_password="${MYSQL_PASSWORD:-root123456}"

mysql \
  --host="$db_host" \
  --port="$db_port" \
  --user="$db_user" \
  --password="$db_password" \
  "$db_name" < "$backup_file"

echo "restored: $backup_file"

