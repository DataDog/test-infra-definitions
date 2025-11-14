ARG CUDA_VERSION=12.4.1
ARG BASE_DISTRO=ubuntu22.04

# Multi-stage image, first we build the CUDA image with the -devel image
FROM nvidia/cuda:${CUDA_VERSION}-devel-${BASE_DISTRO} AS builder

RUN mkdir -p /src
WORKDIR /src
COPY . /src/
RUN make

# Second stage, we build the final image with the runtime image
FROM nvidia/cuda:${CUDA_VERSION}-runtime-${BASE_DISTRO}
COPY --from=builder /src/bin/cuda-basic /usr/local/bin/cuda-basic
