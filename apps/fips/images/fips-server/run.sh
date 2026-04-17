#!/bin/sh

if [ $# -lt 1 ]; then
  echo "Please provide at least one argument ('ecc' or 'rsa') to generate server certificates."
  exit 1
fi


echo "Generating certs ..."
echo "--------------------\n"
./gen_certs.sh $1

shift

echo "Starting server ..."
echo "-------------------\n"
./fips-server server -p 443 $@
