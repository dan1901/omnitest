FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /omnitest ./cmd/omnitest

FROM alpine:3.19

RUN apk --no-cache add ca-certificates

COPY --from=builder /omnitest /usr/local/bin/omnitest

ENTRYPOINT ["omnitest"]
