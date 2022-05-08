FROM golang:1.18.1-alpine AS builder

RUN apk --no-cache add ca-certificates

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -buildvcs=false \
  -ldflags '-extldflags "-static"' \
  -o /go/bin/smtp2workflow


FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/smtp2workflow /smtp2workflow

ENTRYPOINT [ "/smtp2workflow" ]
HEALTHCHECK CMD [ "/smtp2workflow", "-healthcheck" ]

EXPOSE 25/tcp
EXPOSE 465/tcp
