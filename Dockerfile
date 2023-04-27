FROM golang:1.20

WORKDIR /go/src/app
COPY . .

RUN go install ./cmd/main.go

EXPOSE 8080

CMD ["main"]
