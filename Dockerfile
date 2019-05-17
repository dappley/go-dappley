FROM golang:1.11
WORKDIR $GOPATH/src/github.com/dappley/go-dappley
COPY . .
RUN make build

# install and configure tools for cpu/memory metrics
RUN apt-get update && apt-get install -y collectd cron rsyslog
COPY metrics/api/collectd/collectd.conf /etc/collectd/collectd.conf
COPY metrics/api/collectd/remove-collectd-csv.cron /etc/cron.d/metrics-api.cron
RUN crontab /etc/cron.d/metrics-api.cron

CMD ["/bin/bash", "docker-entrypoint.sh"]
