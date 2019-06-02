FROM golang:1.12-alpine as builder

# Create app directory
RUN mkdir -p /usr/src/nanorss
WORKDIR /usr/src/nanorss

# Bundle app source
COPY . /usr/src/nanorss

# Install build dependencies
RUN apk add --no-cache --update build-base git ca-certificates tzdata

# Run tests
RUN go test ./...

# Build app
RUN CGO_ENABLED=0 go build -ldflags="-s -w" && \
  mkdir /usr/src/nanorss/dist && \
  cp -r nanorss-go static templates /usr/src/nanorss/dist

# Copy into a fresh image
FROM scratch

COPY --from=builder /usr/src/nanorss/dist /usr/local/nanorss
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

WORKDIR /usr/local/nanorss

EXPOSE 8080
CMD [ "./nanorss-go" ]
