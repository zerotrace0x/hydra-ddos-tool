# Use the official Golang image as the base image
FROM golang:1.21 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies first
COPY server/go.mod server/go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code to the container
COPY server .

# Build the Go application
RUN go build -o server server.go

# Use a smaller image for the final stage
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the executable, certificates, and key from the builder stage
COPY --from=builder /app/server .
COPY --from=builder /app/cert.pem .
COPY --from=builder /app/key.pem .

# Expose port 8080
EXPOSE 8080

# Set environment variables
ENV API_KEY=mysecretkey
ENV MAX_REQUESTS_PER_MINUTE=10

# Command to run the executable
CMD ["./server"]
```
```dockerfile
# Use the official Nginx image as the base image
FROM nginx:latest

# Set the working directory inside the container
WORKDIR /usr/share/nginx/html

# Remove default content
RUN rm -rf ./*

# Copy the web files to the container
COPY ../../web/index.html .
COPY ../../web/style.css .
COPY ../../web/script.js .

# Command to run Nginx
CMD ["nginx", "-g", "daemon off;"]
```
```dockerfile
# Use the official Nginx image as the base image
FROM nginx:latest

# Copy the Nginx configuration file to the container
COPY nginx.conf /etc/nginx/nginx.conf

# Create the SSL directory
RUN mkdir -p /etc/nginx/ssl

# Copy the ssl files to the container
COPY ../cert.pem /etc/nginx/ssl/cert.pem
COPY ../key.pem /etc/nginx/ssl/key.pem

# Expose ports 80 and 443
EXPOSE 80 443

# Command to run Nginx
CMD ["nginx", "-g", "daemon off;"]
```
```yaml
version: '3.8'

services:
  go-api-server:
    build:
      context: .
      dockerfile: docker/api/Dockerfile
    container_name: go-api-server
    environment:
      - API_KEY=${API_KEY:-mysecretkey}
      - MAX_REQUESTS_PER_MINUTE=${MAX_REQUESTS_PER_MINUTE:-10}
    volumes:
      - ./cert.pem:/app/cert.pem
      - ./key.pem:/app/key.pem
    expose:
      - "8080"
    networks:
      - app-network

  web-frontend:
    build:
      context: .
      dockerfile: docker/frontend/Dockerfile
    container_name: web-frontend
    volumes:
      - ./web/index.html:/usr/share/nginx/html/index.html
      - ./web/style.css:/usr/share/nginx/html/style.css
      - ./web/script.js:/usr/share/nginx/html/script.js
    expose:
      - "80"
    networks:
      - app-network

  nginx:
    build:
      context: ./docker/
      dockerfile: Dockerfile.nginx
    container_name: nginx-reverse-proxy
    ports:
      - "443:443"
      - "80:80"
    volumes:
      - ./cert.pem:/etc/nginx/ssl/cert.pem
      - ./key.pem:/etc/nginx/ssl/key.pem
      - ./docker/nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - go-api-server
      - web-frontend
    networks:
      - app-network

networks:
  app-network:
    driver: bridge
```
```
/server
/web
/docker
```
```
.exe
/server/server
cert.pem
key.pem
.vscode
*.tmp
```
```
upstream api_server {
    server go-api-server:8080;
}

upstream web_server {
    server web-frontend:80;
}

server {
    listen 80;
    server_name localhost;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name localhost;

    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;

    location /api/ {
        proxy_pass http://api_server/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location / {
      proxy_pass http://web_server;
      proxy_set_header Host $host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}