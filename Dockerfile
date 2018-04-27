FROM golang:1.8 AS BUILD

WORKDIR /go/src/app
COPY main.go ./

RUN go get -v -d .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o check_rkn .

ENV LISTEN_ADDR=0.0.0.0:8020

FROM alpine


HEALTHCHECK --interval=30s --timeout=2s \
  CMD curl -fs http://$LISTEN_ADDR/ping

RUN \
    apk add --no-cache --update \
        ca-certificates \
        tzdata \
        curl

COPY --from=BUILD /go/src/app/check_rkn /

ENTRYPOINT ["/check_rkn"]
CMD ["0.0.0.0:8020", "/db"]
VOLUME ["/db"]