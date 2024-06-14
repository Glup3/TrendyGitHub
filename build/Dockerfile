FROM golang:1.22.4-alpine

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/tgh ./cmd/cli

RUN rm -rf /usr/src/app/*

RUN echo '0 * * * * /usr/bin/flock -n /tmp/go_program.lock /usr/local/bin/tgh >/proc/1/fd/1 2>/proc/1/fd/2' > /etc/crontabs/root
CMD ["crond", "-f", "-d", "8"]
