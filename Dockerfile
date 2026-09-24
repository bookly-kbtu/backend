FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/api/main.go -o docs --parseInternal --outputTypes json
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/command ./cmd/command

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H app
WORKDIR /app
COPY --from=build /out/ /usr/local/bin/
COPY --from=build /src/docs ./docs
COPY migrations ./migrations
USER app
EXPOSE 8080
CMD ["api"]
