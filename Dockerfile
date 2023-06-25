# Adapted from https://github.com/pulumi/pulumi-docker-containers/blob/main/docker/pulumi/Dockerfile
# to minimize image size

FROM python:3.10-slim-bullseye AS base

ENV GO_VERSION=1.20.3
ENV GO_SHA=979694c2c25c735755bf26f4f45e19e64e4811d661dd07b8c010f7a8e18adfca

# Install deps all in one step
RUN apt-get update -y && \
  apt-get install -y \
  apt-transport-https \
  build-essential \
  ca-certificates \
  curl \
  git \
  gnupg \
  software-properties-common \
  wget \
  unzip && \
  # Get all of the signatures we need all at once.
  curl -fsSL https://deb.nodesource.com/gpgkey/nodesource.gpg.key  | apt-key add - && \
  curl -fsSL https://dl.yarnpkg.com/debian/pubkey.gpg              | apt-key add - && \
  curl -fsSL https://download.docker.com/linux/debian/gpg          | apt-key add - && \
  curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - && \
  curl -fsSL https://packages.microsoft.com/keys/microsoft.asc     | apt-key add - && \
  # IAM Authenticator for EKS
  curl -fsSLo /usr/bin/aws-iam-authenticator https://github.com/kubernetes-sigs/aws-iam-authenticator/releases/download/v0.5.9/aws-iam-authenticator_0.5.9_linux_amd64 && \
  chmod +x /usr/bin/aws-iam-authenticator && \
  # AWS v2 cli
  curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && \
  unzip -q awscliv2.zip && \
  ./aws/install && \
  rm -rf aws && \
  rm awscliv2.zip && \
  # Add additional apt repos all at once
  echo "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable"      | tee /etc/apt/sources.list.d/docker.list           && \
  echo "deb http://packages.cloud.google.com/apt cloud-sdk-$(lsb_release -cs) main"               | tee /etc/apt/sources.list.d/google-cloud-sdk.list && \
  echo "deb http://apt.kubernetes.io/ kubernetes-xenial main"                                     | tee /etc/apt/sources.list.d/kubernetes.list       && \
  echo "deb [arch=amd64] https://packages.microsoft.com/repos/azure-cli/ $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/azure.list            && \
  # Install second wave of dependencies
  apt-get update -y && \
  apt-get install -y \
  # Pin azure-cli to 2.33.1 as workaround for https://github.com/pulumi/pulumi-docker-containers/issues/106
  "azure-cli=2.33.1-1~bullseye" \
  docker-ce \
  google-cloud-sdk \
  google-cloud-sdk-gke-gcloud-auth-plugin \
  kubectl \
  # xsltproc is required by libvirt-sdk used in the micro-vms scenario
  xsltproc && \
  # Clean up the lists work
  rm -rf /var/lib/apt/lists/*

# Install Go
RUN curl -fsSLo /tmp/go.tgz https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
  echo "${GO_SHA} /tmp/go.tgz" | sha256sum -c - && \
  tar -C /usr/local -xzf /tmp/go.tgz && \
  rm /tmp/go.tgz && \
  export PATH="/usr/local/go/bin:$PATH" && \
  go version
ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

# Install Helm
# Explicitly set env variables that helm reads to their defaults, so that subsequent calls to
# helm will find the stable repo even if $HOME points to something other than /root
# (e.g. in GitHub actions where $HOME points to /github/home).
ENV XDG_CONFIG_HOME=/root/.config
ENV XDG_CACHE_HOME=/root/.cache
RUN curl -L https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash && \
  helm repo add stable https://charts.helm.sh/stable && \
  helm repo update

# Passing --build-arg PULUMI_VERSION=vX.Y.Z will use that version
# of the SDK. Otherwise, we use whatever get.pulumi.com thinks is
# the latest
ARG PULUMI_VERSION

# Install the Pulumi SDK, including the CLI and language runtimes.
RUN curl -fsSL https://get.pulumi.com/ | bash -s -- --version $PULUMI_VERSION && \
  mv ~/.pulumi/bin/* /usr/bin

# Install Pulumi plugins
COPY . /tmp/test-infra
RUN cd /tmp/test-infra && \
  go mod download && \
  export PULUMI_CONFIG_PASSPHRASE=dummy && \
  pulumi --non-interactive plugin install && \
  pulumi --non-interactive plugin ls && \
  cd / && \
  rm -rf /tmp/test-infra

# Install Agent / test requirements
RUN pip3 install -r https://raw.githubusercontent.com/DataDog/datadog-agent-buildimages/main/requirements.txt && \
  go install gotest.tools/gotestsum@latest

# I think it's safe to say if we're using this mega image, we want pulumi
ENTRYPOINT ["pulumi"]
