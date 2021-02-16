FROM alpine:latest as builder

# Install runtime dependencies
RUN apk add --no-cache --update ca-certificates

# Copy build artifact
COPY . /usr/src/nanorss

# Create bin directory
RUN mkdir -p /usr/local/nanorss && \
  cd /usr/src/nanorss && \
  cp nanorss-go /usr/local/nanorss

# Copy into a fresh image
FROM scratch

COPY --from=builder /usr/local/nanorss /usr/local/nanorss
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /usr/local/nanorss

EXPOSE 8080
USER 1001
CMD [ "./nanorss-go" ]
