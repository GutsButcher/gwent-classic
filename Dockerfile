# Use nginx alpine for a lightweight web server
FROM nginx:alpine

# Copy all game files to nginx html directory
COPY . /usr/share/nginx/html/

# Remove the Dockerfile and any unnecessary files from the web directory
RUN rm -f /usr/share/nginx/html/Dockerfile /usr/share/nginx/html/.dockerignore

# Expose port 80
EXPOSE 80

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
