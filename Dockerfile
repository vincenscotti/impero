FROM golang:latest as builder
COPY . /go/src/github.com/vincenscotti/impero/
WORKDIR /go/src/github.com/vincenscotti/impero/
RUN go get -u github.com/valyala/quicktemplate && go get -u github.com/valyala/quicktemplate/qtc
RUN go build

FROM debian:stable
ENV GAME_ADMIN_PW=password GAME_DEBUG_FLAG=true MYSQL_CNX_STRING=impero:password@tcp(172.17.0.4)/impero?parseTime=true&loc=Local
WORKDIR /root/
EXPOSE 8080
COPY --from=builder /go/src/github.com/vincenscotti/impero/impero .
ENTRYPOINT exec /root/impero -pass=${GAME_ADMIN_PW} -debug=${GAME_DEBUG_FLAG}