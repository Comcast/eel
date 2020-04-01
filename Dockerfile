FROM golang:1.8-alpine

ENV BUILD_PACKAGES="build-base git"

WORKDIR /go/src/github.com/Comcast/eel
COPY . ./

RUN apk update && \
    apk upgrade && \
    apk add $BUILD_PACKAGES && \
    go get -u github.com/Comcast/eel && \
    cd test && go test -v && cd .. && \
    go build -o bin/eel

EXPOSE 8080
CMD ./bin/dockereel.sh

