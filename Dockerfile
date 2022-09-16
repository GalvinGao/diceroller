FROM golang:1.16-alpine AS builder

WORKDIR /app
ENV GOOS linux
ENV GOARCH amd64

COPY go.mod ./
COPY go.sum ./

RUN go env -w GO111MODULE=on && go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod download

COPY *.go ./

RUN go build -o /app/diceroller

FROM alpine:latest

ENV TINI_VERSION v0.19.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-static /tini
RUN chmod +x /tini
ENTRYPOINT ["/tini", "--"]

COPY --from=builder /app/diceroller /diceroller
EXPOSE 9726

CMD [ "/diceroller" ]
