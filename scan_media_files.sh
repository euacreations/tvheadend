#!/bin/bash

# Usage: ./scan_media_files.sh <channel_id> <media_directory>
CHANNEL_ID="$1"
LANGUAGE_ID="$2"
MEDIA_DIR="$3"
DB_HOST="localhost"
DB_USER="tvheadend"
DB_PASS="admin123"
DB_NAME="tvheadend"


if [ -z "$CHANNEL_ID" ] || -z "$LANGUAGE_ID" ] || [ -z "$MEDIA_DIR" ]; then
  echo "Usage: $0 <channel_id> <language_id> <media_directory>"
  exit 1
fi

# Function to get file metadata and insert into MySQL
process_file() {
  local file="$1"
  local file_name
  file_name=$(basename "$file")
  local file_path
  file_path=$(realpath --relative-to="$MEDIA_DIR" "$file")
  local duration
  duration=$(ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 "$file" | awk '{print int($1)}')
  local file_size
  file_size=$(stat -c%s "$file")
  local mime_type
  mime_type=$(file --mime-type -b "$file")
  local last_modified
  last_modified=$(stat -c %y "$file" | cut -d'.' -f1)
  local file_hash
  file_hash=$(sha256sum "$file" | awk '{print $1}')
  local scanned_at
  scanned_at=$(date "+%Y-%m-%d %H:%M:%S")

  # Insert or update into MySQL
  mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" <<EOF
INSERT INTO media_files (
  channel_id, file_path, file_name, duration_seconds,language_id, 
  file_size, mime_type, last_modified, file_hash, scanned_at
) VALUES (
  $CHANNEL_ID, '$file_path', '$file_name', $duration,$LANGUAGE_ID,
  $file_size, '$mime_type', '$last_modified', '$file_hash', '$scanned_at'
) ON DUPLICATE KEY UPDATE
  duration_seconds=VALUES(duration_seconds),
  file_size=VALUES(file_size),
  mime_type=VALUES(mime_type),
  last_modified=VALUES(last_modified),
  file_hash=VALUES(file_hash),
  scanned_at=VALUES(scanned_at),
  updated_at=CURRENT_TIMESTAMP;
EOF
}

# Process all files in the directory
find "$MEDIA_DIR" -type f | while read -r file; do
  process_file "$file"
done
