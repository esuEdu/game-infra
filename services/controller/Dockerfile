# ---- build stage ----
FROM golang:1.24-alpine AS build

WORKDIR /src

RUN apk add --no-chache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflag="-s -w" -o /out/controller ./cmd/controller

# ---- runtime stage ----
FROM alpine:3.20

RUN apk add --no-chache ca-certificates git

RUN addgroup -S app && adduser -S app -G app
USER app

WORKDIR /app
COPY --from=build /out/controller ./controller

EXPOSE 8080

ENV HTTP_ADDR=:8080

ENTRYPOINT ["./controller"]