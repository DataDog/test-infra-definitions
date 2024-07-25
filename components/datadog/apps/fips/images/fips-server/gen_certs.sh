#!/bin/bash

set -euo pipefail

function print_usage_and_exit() {
  echo "Usage: $0 <rsa|ecc> [frontend|backend]"
  exit 1
}

if [ $# -lt 1 ]; then
  print_usage_and_exit
fi

if ! command -v openssl &> /dev/null; then
    echo "ERROR! openssl command could not be found and is required to generate the certs!"
    exit 1
fi

openssl_key_type=""
cert_type="$1"
case "$1" in
  ecc)
    openssl ecparam -name secp256r1 > ec_params.tmp
    openssl_key_type="ec:ec_params.tmp"
    ;;
  rsa)
    openssl_key_type="rsa:4096"
    ;;
  *)
    echo "Invalid argument value: $1"
    print_usage_and_exit
    ;;
esac
file_suffix="localhost"

# XXX: LibreSSL on OSX can't use '-addext' so we have to use a workaround to
#      support both OpenSSL and LibreSSL.
#      Issue: https://github.com/libressl-portable/portable/issues/544

cat /etc/ssl/openssl.cnf > openssl.cnf.tmp
printf "[SAN]\nsubjectAltName='DNS:${file_suffix}'" >> openssl.cnf.tmp


openssl req -verbose -newkey "${openssl_key_type}" \
            -x509 \
            -sha256 \
            -days "3650" \
            -nodes \
            -outform "pem" \
            -keyout "${cert_type}_${file_suffix}.key" \
            -out "${cert_type}_${file_suffix}.crt" \
            -config openssl.cnf.tmp \
            -extensions "SAN" \
            -subj "/C=US/ST=NY/L=New York/O=Agent/OU=Agent Core/CN=www.datadoghq.com"

# Remove any temp files
rm -f openssl.cnf.tmp

cat "${cert_type}_${file_suffix}.key" "${cert_type}_${file_suffix}.crt" > "server.pem"

