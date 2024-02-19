FROM golang:1.20-alpine
LABEL AUTHOR=M_Cryptor

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN  go mod download

COPY core ./core
COPY *.go ./

RUN go build -o /app

CMD ["/app/gossip-price"]

EXPOSE 8000
