#!/usr/bin/env sh
set -f

[ -n "$JAVA_OPTS" ] || JAVA_OPTS="-Xmx128M -Xms128M"
[ -n "$RMI_PORT" ] || RMI_PORT="9010"
[ -n "$HOST_NAME" ] || HOST_NAME=`awk 'END{print $1}' /etc/hosts`
[ -n "$SSL_MODE" ] || SSL_MODE="false"

echo "Using `java --version`"
echo "With JAVA_OPTS '${JAVA_OPTS}'"

# shellcheck disable=SC2086
javac -d app SimpleApp.java

echo "Starting app with hostname set to ${HOST_NAME}"
echo "RMI port is set to ${RMI_PORT}"
echo "SSL: ${SSL_MODE}"

java -cp ./app \
    ${JAVA_OPTS} \
    -Dcom.sun.management.jmxremote=true \
    -Dcom.sun.management.jmxremote.port=${RMI_PORT} \
    -Dcom.sun.management.jmxremote.rmi.port=${RMI_PORT} \
    -Dcom.sun.management.jmxremote.authenticate=false \
    -Dcom.sun.management.jmxremote.ssl=${SSL_MODE} \
    -Dcom.sun.management.jmxremote.registry.ssl=${SSL_MODE} \
    -Djava.rmi.server.hostname=${HOST_NAME} \
    SimpleApp

# java -jar jmxterm-1.0.2-uber.jar -l service:jmx:rmi:///jndi/rmi://${HOST_NAME}:${RMI_PORT}"/jmxrmi
