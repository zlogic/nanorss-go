FROM golang:1.12-alpine as builder

# Create app directory
RUN mkdir -p /usr/src/nanorss
WORKDIR /usr/src/nanorss

# Bundle app source
COPY . /usr/src/nanorss

# Install build dependencies
RUN apk add --no-cache --update build-base git

# Run tests
RUN go test ./...

# Build app
RUN go build && \
  mkdir /usr/src/nanorss/dist && \
  cp -r nanorss-go static templates /usr/src/nanorss/dist

# Copy into a fresh image
FROM alpine:3.8

COPY --from=builder /usr/src/nanorss/dist /usr/local/nanorss

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /usr/local/nanorss

EXPOSE 8080
CMD [ "./nanorss-go" ]
