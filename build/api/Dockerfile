FROM golang:1.26-alpine AS build

WORKDIR /src

# Bog'liqliklar alohida qatlamda: kod o'zgarganda `go mod download`
# qayta ishlamaydi.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 — statik binary, shuning uchun yakuniy image'da
# libc ham kerak emas. -trimpath yo'llarni olib tashlaydi (build
# mashinasining kataloglari binary ichida qolmasin).
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.21

# ca-certificates SHART: SMTP TLS va Telegram Bot API HTTPS ustidan
# ishlaydi. Usiz "x509: certificate signed by unknown authority".
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 dehqon

USER dehqon
COPY --from=build /out/api /usr/local/bin/api

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
