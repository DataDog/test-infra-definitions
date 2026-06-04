FROM centos:7

ENV DOCKER_DD_AGENT=yes

# import all gpg keys
RUN rpm --import /etc/pki/rpm-gpg/RPM-GPG-KEY-*

# use vault repository
RUN mv /etc/yum.repos.d/CentOS-Base.{repo,bak}
COPY vault.repo.* /etc/yum.repos.d/
RUN mv /etc/yum.repos.d/vault.repo{.$(uname -m),}

RUN mkdir -p /opt/datadog-agent/embedded/bin
RUN yum -y install xfsprogs e2fsprogs iproute perl && \
    yum clean all && \
    rm -rf /var/cache/yum
