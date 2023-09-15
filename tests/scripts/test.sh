#!/usr/bin/env bash
set -e

CONTENT_TYPE="Content-Type: image/jpeg"

# --== File Upload Test ==--

printf "Testing file upload: "

IMAGELOC="$(
  curl -s -D - -F 'file=@/tmp/test.jpg' grombley:3000/upload |
  grep Location |
  awk '{print $2}' |
  tr -d '\r'
)"

curl --fail-with-body -s -I "http://grombley:3000${IMAGELOC}" | tee > /tmp/file

if grep -q "$CONTENT_TYPE" /tmp/file; then
  printf "✅ - File upload success\n\n"
else
  printf "❌ - File upload failed\n\n"
  cat /tmp/file
  exit 1
fi

# --== URL Upload Test ==--

printf "Testing URL upload: "

URLPAYLOAD='{"url":"http://nginx/test.test.jpg"}'

IMAGELOC=$(
  curl -s -D - -H "Content-Type: application/json" -d "${URLPAYLOAD}" grombley:3000/url |
  grep Location |
  awk '{print $2}' |
  tr -d '\r'
)

curl --fail-with-body -s -I "grombley:3000${IMAGELOC}" | tee > /tmp/url

if grep -q "$CONTENT_TYPE" /tmp/url; then
  printf "✅ - URL upload success\n\n"
else
  printf "❌ - URL upload failed\n\n"
  cat /tmp/url
  exit 1
fi

