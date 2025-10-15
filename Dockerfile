# multi-stage
FROM golang:1.23 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/library ./


FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /bin/library /bin/library
COPY static ./static
ENV HTTP_ADDR=:8080
ENV DB_DSN=postgres://library:library@db:5432/library?sslmode=disable
EXPOSE 8080
ENTRYPOINT ["/bin/library"]