FROM golang:1.21-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o api-notificaciones-go .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /app/api-notificaciones-go ./api-notificaciones-go
COPY --from=build /app/templates ./templates
COPY --from=build /app/openapi.yaml ./openapi.yaml
EXPOSE 4000
CMD ["./api-notificaciones-go"]
