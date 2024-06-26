# --== build image ==--
FROM golang:alpine as BUILDER

WORKDIR /build

COPY go.mod go.sum /build/

RUN go mod download

COPY . /build

RUN go build

#--== final image ==--
FROM alpine

COPY --from=BUILDER /build/image-uploader /opt/grombley/

WORKDIR /opt/grombley

CMD ./image-uploader
