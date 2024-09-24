FROM golang:1.22.0-alpine as builder
RUN apk add --no-cache git make
ENV GOOS=linux
ENV CGO_ENABLED=0
ENV GO111MODULE=on
COPY . /src
WORKDIR /src
RUN rm -f go.sum
RUN go get ./...
RUN make release

FROM alpine:3.19.1
RUN apk add --no-cache ca-certificates curl busybox-extras
WORKDIR /app
COPY --from=builder /src/bin/troll /app/troll
ENTRYPOINT ["/app/troll"]
