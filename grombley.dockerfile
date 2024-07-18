# --== build image ==--
FROM golang:alpine AS builder

WORKDIR /build

COPY go.mod go.sum /build/

RUN go mod download

COPY . /build

RUN go build

#--== final image ==--
FROM alpine

COPY --from=builder /build/image-uploader /opt/grombley/
COPY --from=builder /build/templates /opt/grombley/templates

WORKDIR /opt/grombley

CMD ["./image-uploader"]
