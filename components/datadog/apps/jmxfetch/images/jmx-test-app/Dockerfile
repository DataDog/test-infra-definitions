# syntax=docker/dockerfile:1

# Allows to cheange the JDK image used
ARG JRE_DOCKER_IMAGE=eclipse-temurin:11

# Use the official JDK image as the base image
FROM ${JRE_DOCKER_IMAGE}

WORKDIR /app

COPY run.sh SimpleApp.java /app/

EXPOSE 9010
EXPOSE 8080

ENTRYPOINT [ "/app/run.sh" ]
