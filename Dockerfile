FROM golang:1.23 AS builder
ENV CGO_ENABLED=0
ENV GOOS=linux
RUN apt update && apt install -y git make
WORKDIR /go/src/tvhtc2
COPY go.* ./
RUN go mod download
COPY . .
RUN make

FROM alpine:3
RUN mkdir /srv/tvhtc2 /etc/tvhtc2 \
    && addgroup -g 1000 dane \
    && adduser -u 1000 -h /tmp -S -D -H -G dane dane \
    && chown -R dane:dane /srv/tvhtc2 \
    && apk add --no-cache ffmpeg socat netcat-openbsd
VOLUME /etc/tvhtc2
VOLUME /srv/tvhtc2
COPY --from=builder /go/src/tvhtc2/bin/tvhtc2 /usr/bin/tvhtc2
COPY --from=builder /go/src/tvhtc2/bin/tvhtc2-client /usr/bin/tvhtc2-client
COPY --from=builder /go/src/tvhtc2/bin/tvhtc2-renamer /usr/bin/tvhtc2-renamer
COPY --chmod=755 docker/startup.sh /startup.sh
USER dane:dane
WORKDIR /tmp
ENTRYPOINT ["/startup.sh"]
