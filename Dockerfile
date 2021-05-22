FROM golang:1.8-alpine AS builder

ENV BUILD_PACKAGES="build-base git"

WORKDIR /go/src/github.com/Comcast/eel
COPY . ./

RUN apk update && \
    apk upgrade && \
    apk add $BUILD_PACKAGES && \
    go get -u github.com/Comcast/eel && \
    cd test && go test -v -cover && cd .. && \
    go build -o bin/eel


FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata curl

WORKDIR /go/src/github.com/Comcast/eel
COPY --from=builder /go/src/github.com/Comcast/eel/bin ./bin
COPY --from=builder /go/src/github.com/Comcast/eel/config-eel ./config-eel
COPY --from=builder /go/src/github.com/Comcast/eel/config-handlers ./config-handlers
COPY --from=builder /go/src/github.com/Comcast/eel/test/data ./test/data

EXPOSE 8080

CMD ./bin/dockereel.sh
