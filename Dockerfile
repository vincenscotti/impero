FROM golang:latest as builder
RUN go get -u github.com/golang/dep/cmd/dep && go get -u github.com/valyala/quicktemplate && go get -u github.com/valyala/quicktemplate/qtc
COPY . /go/src/github.com/vincenscotti/impero/
WORKDIR /go/src/github.com/vincenscotti/impero/templates
RUN qtc
WORKDIR /go/src/github.com/vincenscotti/impero
RUN dep ensure && go build

FROM debian:stable
WORKDIR /root/
EXPOSE 8080
COPY --from=builder /go/src/github.com/vincenscotti/impero/impero .
COPY --from=builder /go/src/github.com/vincenscotti/impero/static/ ./static/
