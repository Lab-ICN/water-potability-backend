FROM golang:1.25-alpine AS builder

WORKDIR /tmp/build
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -o server ./cmd/mqtt

FROM gcr.io/distroless/static-debian12
COPY --from=builder /tmp/build/server /

CMD ["/server"]
