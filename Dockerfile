# ---- Build Stage ----
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -tags production -ldflags="-s -w" -o dfkgo main.go

# ---- Runtime Stage ----
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /app/dfkgo .

EXPOSE 8888

CMD ["./dfkgo"]
