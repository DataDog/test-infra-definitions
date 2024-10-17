# Adapted from https://github.com/pulumi/pulumi-docker-containers/blob/main/docker/pulumi/Dockerfile
# to minimize image size

FROM public.ecr.aws/docker/library/python:3.13-slim-bullseye AS base

ENV GO_VERSION=1.22.8
ENV GO_SHA=5f467d29fc67c7ae6468cb6ad5b047a274bae8180cac5e0b7ddbfeba3e47e18f
ENV HELM_VERSION=3.12.3
ENV HELM_SHA=1b2313cd198d45eab00cc37c38f6b1ca0a948ba279c29e322bdf426d406129b5
ARG CI_UPLOADER_SHA=873976f0f8de1073235cf558ea12c7b922b28e1be22dc1553bf56162beebf09d
ARG CI_UPLOADER_VERSION=2.30.1
# Skip Pulumi update warning https://www.pulumi.com/docs/cli/environment-variables/
ENV PULUMI_SKIP_UPDATE_CHECK=true

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
  curl --retry 10 -fsSL https://deb.nodesource.com/gpgkey/nodesource.gpg.key    | apt-key add - && \
  curl --retry 10 -fsSL https://dl.yarnpkg.com/debian/pubkey.gpg                | apt-key add - && \
  curl --retry 10 -fsSL https://download.docker.com/linux/debian/gpg            | apt-key add - && \
  curl --retry 10 -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg   | apt-key add - && \
  curl --retry 10 -fsSL https://packages.microsoft.com/keys/microsoft.asc       | apt-key add - && \
  curl --retry 10 -fsSL https://pkgs.k8s.io/core:/stable:/v1.28/deb/Release.key | gpg --dearmor -o /usr/share/keyrings/kubernetes-archive-keyring.gpg && \
  curl --retry 10 -fsSL https://apt.releases.hashicorp.com/gpg                  | gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg && \
  # IAM Authenticator for EKS
  curl --retry 10 -fsSLo /usr/bin/aws-iam-authenticator https://github.com/kubernetes-sigs/aws-iam-authenticator/releases/download/v0.5.9/aws-iam-authenticator_0.5.9_linux_amd64 && \
  chmod +x /usr/bin/aws-iam-authenticator && \
  # AWS v2 cli
  curl --retry 10 -fsSLo awscliv2.zip https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip && \
  unzip -q awscliv2.zip && \
  ./aws/install && \
  rm -rf aws && \
  rm awscliv2.zip && \
  # Add additional apt repos all at once
  echo "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable"                                          | tee /etc/apt/sources.list.d/docker.list           && \
  echo "deb https://packages.cloud.google.com/apt cloud-sdk-$(lsb_release -cs) main"                                                  | tee /etc/apt/sources.list.d/google-cloud-sdk.list && \
  echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.28/deb/ /"            | tee /etc/apt/sources.list.d/kubernetes.list       && \
  echo "deb [arch=amd64] https://packages.microsoft.com/repos/azure-cli/ $(lsb_release -cs) main"                                     | tee /etc/apt/sources.list.d/azure.list            && \
  echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/hashicorp.list        && \
  # Install second wave of dependencies
  apt-get update -y && \
  apt-get install -y \
  azure-cli \
  docker-ce \
  google-cloud-sdk \
  google-cloud-sdk-gke-gcloud-auth-plugin \
  jq \
  kubectl \
  vault \
  # xsltproc is required by libvirt-sdk used in the micro-vms scenario
  xsltproc && \
  # Remove the cap_ipc_lock capability from vault https://github.com/hashicorp/vault/issues/10924
  setcap -r /usr/bin/vault && \
  # Install the datadog-ci-uploader
  curl --retry 10 -fsSL https://github.com/DataDog/datadog-ci/releases/download/v${CI_UPLOADER_VERSION}/datadog-ci_linux-x64 --output "/usr/local/bin/datadog-ci" && \
  echo "${CI_UPLOADER_SHA} /usr/local/bin/datadog-ci" | sha256sum --check && \
  chmod +x /usr/local/bin/datadog-ci && \
  # Clean up the lists work
  rm -rf /var/lib/apt/lists/*

# Install Go
RUN curl --retry 10 -fsSLo /tmp/go.tgz https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
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
RUN curl --retry 10 -fsSLo /tmp/helm.tgz https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz && \
  echo "${HELM_SHA} /tmp/helm.tgz" | sha256sum -c - && \
  tar -C /usr/local/bin -xzf /tmp/helm.tgz --strip-components=1 linux-amd64/helm && \
  rm /tmp/helm.tgz && \
  helm version && \
  helm repo add stable https://charts.helm.sh/stable && \
  helm repo update

# Passing --build-arg PULUMI_VERSION=vX.Y.Z will use that version
# of the SDK. Otherwise, we use whatever get.pulumi.com thinks is
# the latest
ARG PULUMI_VERSION

# Install the Pulumi SDK, including the CLI and language runtimes.
RUN --mount=type=secret,id=github_token \
  export GITHUB_TOKEN=$(cat /run/secrets/github_token) && \
  curl --retry 10 -fsSL https://get.pulumi.com/ | bash -s -- --version $PULUMI_VERSION && \
  mv ~/.pulumi/bin/* /usr/bin

# Install Pulumi plugins
# The time resource is installed explicitly here instead in go.mod
# because it's not used directly by this repository, thus go mod tidy
# would remove it...
COPY . /tmp/test-infra
RUN --mount=type=secret,id=github_token \
  export GITHUB_TOKEN=$(cat /run/secrets/github_token) && \
  cd /tmp/test-infra && \
  go mod download && \
  export PULUMI_CONFIG_PASSPHRASE=dummy && \
  pulumi --logflow --logtostderr -v 5 --non-interactive plugin install && \
  pulumi --non-interactive plugin ls && \
  cd /

# Install Agent requirements, required to run invoke tests task
# Remove AWS-related deps as we already install AWS CLI v2
RUN pip3 install -r https://raw.githubusercontent.com/DataDog/datadog-agent-buildimages/main/requirements/e2e.txt & \
  pip3 install -r /tmp/test-infra/requirements.txt & \
  go install gotest.tools/gotestsum@latest

RUN rm -rf /tmp/test-infra

# Configure aws retries
COPY .awsconfig $HOME/.aws/config
