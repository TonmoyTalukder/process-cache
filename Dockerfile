FROM golang:1.24 AS test

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN go test ./...

CMD ["sh", "-c", "go test ./... && go test -race ./... && go test -bench=. -benchmem ./..."]
