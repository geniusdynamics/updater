FROM golang:1.20-alpine

WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .

RUN go build -o /usr/local/bin/updater ./cmd/cli

ENTRYPOINT ["/usr/local/bin/updater"]