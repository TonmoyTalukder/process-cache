FROM golang:1.24 AS test

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

CMD ["sh", "-c", "go test ./... && go test -race ./... && go test -bench=. -benchmem ./..."]
