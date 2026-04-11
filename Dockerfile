FROM golang:1.25 as builder
WORKDIR /code
ADD go.mod go.sum /code/
RUN go mod download
ADD . .
RUN go build -o /main  cmd/main/main.go
FROM gcr.io/distroless/base
WORKDIR /
COPY --from=builder /main /usr/bin/main
ENTRYPOINT ["main"]