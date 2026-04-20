FROM golang:1.23 AS builder
WORKDIR /app
COPY . .
RUN go mod download

RUN go build -o node

ENV VOLUME_PATH="/volume/"
ENV RABBITMQ_HOSTNAME_ENV="dag-ssle-rabbitmq"
ENV WS_HOSTNAME_ENV="dag-ssle-bootstrap-server"

CMD ["sh", "start.sh", "dag-ssle-rabbitmq", "15672", "./node"]