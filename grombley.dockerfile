# --== build image ==--
FROM golang:alpine as BUILDER

WORKDIR /build

COPY . /build

RUN go build

#--== final image ==--
FROM alpine

COPY --from=BUILDER /build/config.toml .
COPY --from=BUILDER /build/image-uploader .

CMD /image-uploader
