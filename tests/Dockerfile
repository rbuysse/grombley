FROM alpine

RUN apk update \
 && apk add \
    curl

COPY tests/scripts/* /tmp/
COPY tests/images/* /tmp/
