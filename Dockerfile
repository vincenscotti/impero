FROM golang:latest as builder
RUN go get -u github.com/golang/dep/cmd/dep && go get -u github.com/valyala/quicktemplate/qtc
COPY . /go/src/github.com/vincenscotti/impero/
WORKDIR /go/src/github.com/vincenscotti/impero/templates
RUN qtc
WORKDIR /go/src/github.com/vincenscotti/impero
RUN dep ensure && go build
WORKDIR /go/src/github.com/vincenscotti/impero/jpgtomap
RUN go build
WORKDIR /go/src/github.com/vincenscotti/impero/maps
RUN ./map.sh

FROM debian:stable
RUN apt-get update && apt-get install -y ca-certificates
WORKDIR /root/
EXPOSE 8080
COPY --from=builder /go/src/github.com/vincenscotti/impero/impero .
COPY --from=builder /go/src/github.com/vincenscotti/impero/static/ ./static/
COPY --from=builder /go/src/github.com/vincenscotti/impero/map.sql ./map.sql
