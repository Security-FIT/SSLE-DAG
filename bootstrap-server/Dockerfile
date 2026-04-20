FROM node:alpine

# Set working directory
WORKDIR /app

# Copy content of the web application to the container
COPY . .

# Install all dependencies and build the application
RUN yarn install

ENV VOLUME_PATH="/volume/"

# Serve static web application using serve module
CMD ["yarn", "start"]
