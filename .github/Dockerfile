FROM alpine:latest as builder

# Create non-root user
RUN adduser -D -H -u 10001 nanorss

# Install runtime dependencies
RUN apk add --no-cache --update ca-certificates tzdata

# Copy build artifact
COPY . /usr/src/nanorss

# Create bin directory
RUN mkdir -p /usr/local/nanorss && \
  cd /usr/src/nanorss && \
  cp -r nanorss-go static templates /usr/local/nanorss

# Copy into a fresh image
FROM scratch

COPY --from=builder /usr/local/nanorss /usr/local/nanorss
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd

WORKDIR /usr/local/nanorss

EXPOSE 8080
USER nanorss
CMD [ "./nanorss-go" ]
