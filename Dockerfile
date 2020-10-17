FROM golang:1.15.2-alpine3.12 as builder

WORKDIR /workdir

ENV GO111MODULE=on
COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN GOOS=linux go build -mod=readonly  -v  -o /go-app main.go

FROM alpine:3.12
COPY --from=builder /go-app .
ENTRYPOINT ["./go-app"]
