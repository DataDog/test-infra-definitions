FROM nginx:stable-alpine-slim

COPY nginx.conf /etc/nginx/nginx.conf
COPY index.html /etc/nginx/html/index.html

VOLUME /var/cache/nginx
VOLUME /var/run

EXPOSE 80
