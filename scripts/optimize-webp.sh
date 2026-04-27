#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/optimize-webp.sh DIRECTORY

Example:
  scripts/optimize-webp.sh "/path/to/archive"

Recursively re-encode every .webp file under DIRECTORY with ffmpeg, replacing
each original image after a successful encode.

Optional environment variables:
  WEBP_QUALITY             Quality from 0-100. Default: 80
  WEBP_COMPRESSION_LEVEL   Compression effort from 0-6. Default: 6
  WEBP_MIN_BYTES           Minimum file size to optimize. Default: 204800
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -ne 1 ]]; then
  usage >&2
  exit 2
fi

target_dir=$1
quality=${WEBP_QUALITY:-80}
compression_level=${WEBP_COMPRESSION_LEVEL:-6}
min_bytes=${WEBP_MIN_BYTES:-204800}

if ! command -v ffmpeg >/dev/null 2>&1; then
  echo "error: ffmpeg is not installed or not on PATH" >&2
  exit 1
fi

if [[ ! -d "$target_dir" ]]; then
	echo "error: directory does not exist: $target_dir" >&2
	exit 1
fi

target_dir=$(cd "$target_dir" && pwd -P)

processed=0
failed=0
skipped=0

echo "scanning: $target_dir"

while IFS= read -r -d '' image_path; do
	image_dir=$(dirname "$image_path")
	image_name=$(basename "$image_path")
	temp_path="$image_dir/.optimize-webp.$$.$RANDOM.$image_name"

	if [[ ! -f "$image_path" ]]; then
		echo "skipped missing file: $image_path" >&2
		skipped=$((skipped + 1))
		continue
	fi

	image_size=$(stat -c '%s' "$image_path")
	if (( image_size <= min_bytes )); then
		skipped=$((skipped + 1))
		echo "skipped under threshold: $image_path"
		continue
	fi

	if [[ -e "$temp_path" ]]; then
		echo "skipped temp path collision: $image_path" >&2
		skipped=$((skipped + 1))
		continue
	fi

	if ffmpeg \
    -hide_banner \
    -loglevel error \
    -y \
    -i "$image_path" \
    -map 0:v:0 \
    -c:v libwebp \
    -quality "$quality" \
    -compression_level "$compression_level" \
    -preset picture \
    -an \
    "$temp_path"; then
    if [[ ! -s "$temp_path" ]]; then
      echo "skipped empty output: $image_path" >&2
      rm -f "$temp_path"
      skipped=$((skipped + 1))
      continue
    fi

    chmod --reference="$image_path" "$temp_path" 2>/dev/null || true
    mv "$temp_path" "$image_path"
    processed=$((processed + 1))
    echo "optimized: $image_path"
  else
    echo "failed: $image_path" >&2
    rm -f "$temp_path"
    failed=$((failed + 1))
  fi
done < <(find "$target_dir" -type f -iname '*.webp' ! -name '.optimize-webp.*' -print0)

echo "done: $processed optimized, $skipped skipped, $failed failed"

if [[ $failed -gt 0 ]]; then
  exit 1
fi
