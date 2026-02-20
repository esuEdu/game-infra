# ---- build stage ----
FROM golang:1.24-alpine AS build

WORKDIR /src

# RUN apk add --no-chache ca-certificates git

COPY go.mod ./
RUN go mod download

COPY . .

# Build a static-ish binary (alpine-friendly)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o /out/controller ./cmd/controller

# ---- runtime stage ----
FROM alpine:3.20

RUN apk add --no-cache \
    ca-certificates \
    git

WORKDIR /app
COPY --from=build /out/controller ./controller

EXPOSE 8080

ENV HTTP_ADDR=:8080

ENTRYPOINT ["./controller"]
