#!/usr/bin/env bash
set -e

CONTENT_TYPE="Content-Type: image/jpeg"

# --== File Upload Test ==--

printf "Testing file upload: "

IMAGELOC="$(curl -s -F 'file=@/tmp/test.jpg' grombley:3000/upload)"

curl --fail-with-body -s -I "$IMAGELOC" | tee > /tmp/file

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
  curl -s \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -d "${URLPAYLOAD}" grombley:3000/url \
      | grep -oP '(?<="url":")[^"]+'
)

curl --fail-with-body -s -I "$IMAGELOC" | tee > /tmp/url

if grep -q "$CONTENT_TYPE" /tmp/url; then
  printf "✅ - URL upload success\n\n"
else
  printf "❌ - URL upload failed\n\n"
  cat /tmp/url
  exit 1
fi

