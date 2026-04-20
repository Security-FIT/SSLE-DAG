FROM node:alpine

# Set working directory
WORKDIR /app

# Copy content of the web application to the container
COPY . .

# Install all dependencies and build the application
RUN yarn install
RUN yarn build
RUN npm install -g serve

# Expose port to public
EXPOSE 3000

# Serve static web application using serve module
CMD ["serve", "-s", "/app/build", "-l", "tcp://0.0.0.0:3000"]
