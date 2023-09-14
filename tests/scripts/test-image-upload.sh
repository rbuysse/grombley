#!/usr/bin/env sh
# not standalone, runs in Docker

IMAGELOC=$( \
   curl -s -D - -F 'file=@/tmp/test.jpg' grombley:3000/upload | \
   grep Location | \
   awk '{print $2}') \
&& curl --fail-with-body -s -I "grombley:3000${IMAGELOC}"
