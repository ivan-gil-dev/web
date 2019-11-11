FROM golang:alpine3.10

RUN mkdir /program
ADD . /program
WORKDIR /program
RUN apk update && apk add git
RUN go get "github.com/GeertJohan/go.rice"
RUN go get "github.com/lib/pq"
RUN go get "github.com/tidwall/gjson"
RUN go build -o main .
CMD [ "/program/main" ]