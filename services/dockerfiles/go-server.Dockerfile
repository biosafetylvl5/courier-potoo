FROM golang:latest

WORKDIR /app

COPY server/go.mod ./

RUN go mod download

COPY server/*.go ./
COPY server/pages ./pages

RUN CGO_ENABLED=0 GOOS=linux go build -o /go-server

CMD ["/go-server"]
