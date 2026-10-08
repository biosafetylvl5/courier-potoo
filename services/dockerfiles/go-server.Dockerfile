FROM golang:latest

WORKDIR /app

COPY server/go.mod server/go.sum ./

RUN go mod download

COPY server/*.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /go-server

CMD ["/go-server"]
