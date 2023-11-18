FROM golang:1.21 AS builder
ENV CGO_ENABLED=0
ENV GOOS=linux
RUN apt update && apt install -y git make
WORKDIR /go/src/tvhtc2
COPY go.* ./
RUN go mod download
COPY . .
RUN make

FROM alpine:3
COPY --from=builder /go/src/tvhtc2/bin/tvhtc2 /usr/bin/tvhtc2
COPY --from=builder /go/src/tvhtc2/bin/tvhtc2-client /srv/tvhtc2/tvhtc2-client
COPY --from=builder /go/src/tvhtc2/bin/tvhtc2-renamer /usr/bin/tvhtc2-renamer
RUN addgroup -g 1002 -S smbw \
    && adduser -u 1000 -h /tmp -S -D -H -G smbw dane \
    && chown -R dane:smbw /srv/tvhtc2 \
    && apk add --no-cache ffmpeg socat netcat-openbsd
USER dane:smbw
WORKDIR /tmp
VOLUME /etc/tvhtc2
ENTRYPOINT ["/usr/bin/tvhtc2"]
