FROM golang:1.22.4-alpine

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/tgh ./cmd/cli

RUN rm -rf /usr/src/app/*

RUN echo '*/2 * * * * /usr/bin/flock -n /tmp/tgh_daily.lock /usr/local/bin/tgh search >/proc/1/fd/1 2>/proc/1/fd/2' > /etc/crontabs/root \
  # && echo '*/2 0-14,17-23 * * * /usr/bin/flock -n /tmp/tgh_history.lock /usr/local/bin/tgh history >/proc/1/fd/1 2>/proc/1/fd/2' >> /etc/crontabs/root \
  # && echo '*/2 * * * * /usr/bin/flock -n /tmp/tgh_history_40k.lock /usr/local/bin/tgh history-40k >/proc/1/fd/1 2>/proc/1/fd/2' >> /etc/crontabs/root \
  && echo '*/5 * * * * /usr/bin/flock -n /tmp/tgh_repair_40k.lock /usr/local/bin/tgh repair-40k >/proc/1/fd/1 2>/proc/1/fd/2' >> /etc/crontabs/root \
  && echo '59 * * * * /usr/bin/flock -n /tmp/tgh_reset.lock /usr/local/bin/tgh reset >/proc/1/fd/1 2>/proc/1/fd/2' >> /etc/crontabs/root \
  && echo '0 * * * * /usr/bin/flock -n /tmp/tgh_history_refresh.lock /usr/local/bin/tgh refresh >/proc/1/fd/1 2>/proc/1/fd/2' >> /etc/crontabs/root

CMD ["crond", "-f", "-d", "8"]
