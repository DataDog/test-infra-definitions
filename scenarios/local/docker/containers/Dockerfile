FROM geerlingguy/docker-ubuntu2204-ansible:latest

COPY allow_root.conf /etc/ssh/sshd_config.d/allow_root.conf

RUN set -eu && \
    apt-get update; \
    apt-get install -y openssh-server; \
    echo 'root:root123' | chpasswd;
