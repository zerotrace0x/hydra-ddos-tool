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