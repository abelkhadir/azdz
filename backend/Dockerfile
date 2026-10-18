FROM golang:1.27-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /social-network ./server.go

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /social-network ./social-network
COPY internal/pkg/db/migrations ./internal/pkg/db/migrations
RUN mkdir -p /app/data /app/uploads/posts /app/uploads/avatars

ENV PORT=8080
ENV DATABASE_PATH=/app/data/social-network.db
EXPOSE 8080
CMD ["./social-network"]
