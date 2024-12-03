FROM golang:1.23.3-alpine

WORKDIR /usr/src/app

COPY . .

RUN go build -v -o /usr/local/bin/app ./...

CMD ["app"]

