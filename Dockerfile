FROM golang:1.25-alpine AS builder

WORKDIR /app
ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0
ARG VERSION=dev

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -trimpath -ldflags="-s -w -X main.Version=${VERSION}" -o /app/diceroller

FROM alpine:latest

ENV TINI_VERSION=v0.19.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-static /tini
RUN chmod +x /tini
ENTRYPOINT ["/tini", "--"]

COPY --from=builder /app/diceroller /diceroller
EXPOSE 9726

CMD ["/diceroller"]
