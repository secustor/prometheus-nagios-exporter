# Step 1: Install CA certificates and setup Go binary build
FROM alpine AS build

RUN apk add --update --no-cache git gcc musl-dev ca-certificates

RUN addgroup -S service && adduser -D -G service service

# Step 2: Copy binaries and ca-certificates to scratch (empty) image
FROM scratch

COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY target/prometheus-nagios-exporter /bin/

USER service

ENV PATH="/bin"

ARG BUILD_DATE
ARG BUILD_NUMBER
ARG VCS_SHA

LABEL maintainer="devops@itsdone.at" \
    com.ft.build-number="$BUILD_NUMBER" \
    org.opencontainers.authors="devops@itsdone.at" \
    org.opencontainers.created="$BUILD_DATE" \
    org.opencontainers.licenses="MIT" \
    org.opencontainers.revision="$VCS_SHA" \
    org.opencontainers.title="prometheus-nagios-exporter" \
    org.opencontainers.source="https://gitlab.itsdone.at/devops/prometheus-nagios-exporter" \
    org.opencontainers.vendor="itsdone"

ENTRYPOINT ["prometheus-nagios-exporter"]
