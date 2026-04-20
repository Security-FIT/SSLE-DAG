FROM node:alpine

# Set working directory
WORKDIR /app

# Copy content of the web application to the container
COPY . .

RUN apk update && apk add --no-cache curl

# Install all dependencies and build the application
RUN yarn install
RUN yarn build

ENV VOLUME_PATH="/volume/"
ENV RABBITMQ_URL="amqp://dag-ssle-rabbitmq"

# Serve static web application using serve module
CMD ["sh", "start.sh", "dag-ssle-rabbitmq", "15672", "yarn", "start:prod"]
