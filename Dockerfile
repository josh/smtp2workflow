FROM golang:1.17.8-alpine AS builder

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags '-extldflags "-static"' \
  -o /go/bin/smtp2workflow


FROM scratch
COPY --from=builder /go/bin/smtp2workflow /smtp2workflow

ENTRYPOINT [ "/smtp2workflow" ]
HEALTHCHECK CMD [ "/smtp2workflow", "-healthcheck" ]

EXPOSE 25/tcp
EXPOSE 465/tcp
