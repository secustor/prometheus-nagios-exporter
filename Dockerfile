FROM golang:1.10-alpine AS build

WORKDIR /go/src/github.com/Financial-Times/prometheus-nagios-exporter/

RUN apk add --update --no-cache curl git && \
    curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

COPY Gopkg.* ./
COPY cmd/ cmd/
COPY internal/ internal/

RUN dep ensure
RUN go build -o /tmp/nagios-exporter cmd/nagios-exporter/main.go

FROM alpine:latest

RUN apk add --update --no-cache ca-certificates

WORKDIR /root/

COPY --from=build /tmp/nagios-exporter .

EXPOSE 9942

ENTRYPOINT ["/root/nagios-exporter"]
