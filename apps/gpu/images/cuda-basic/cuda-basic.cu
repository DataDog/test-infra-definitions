// Sample from NVIDIA:
// https://github.com/NVIDIA/cuda-samples/blob/master/Samples/0_Introduction/vectorAdd/vectorAdd.cu

#include <stdexcept>
#include <stdio.h>
#include <string>
#include <unistd.h>

// For the CUDA runtime routines (prefixed with "cuda_")
#include <cuda_runtime.h>

// Code to be executed in the GPU. Allows managing the number of loops to have
// an increased execution time.
__global__ void vectorSumKernel(const float *A, const float *B, float *C,
								int numElements, int loops) {
	int i = blockDim.x * blockIdx.x + threadIdx.x;

	for (size_t loopIdx = 0; loopIdx < loops; loopIdx++) {
		if (i < numElements) {
			C[i] = A[i] + B[i] + 0.0f;
		}
	}
}

int main(int argc, const char **argv) {
	// Error code to check return values for CUDA calls
	cudaError_t err = cudaSuccess;

	if (argc != 4) {
		fprintf(stderr, "Usage: %s <numElements> <loops> <waitTimeSeconds>\n",
				argv[0]);
		exit(EXIT_FAILURE);
	}

	int numElements, loops, waitTimeSeconds;

	try {
		numElements = std::stoi(argv[1]);
		loops = std::stoi(argv[2]);
		waitTimeSeconds = std::stoi(argv[3]);
	} catch (const std::invalid_argument &e) {
		fprintf(stderr, "Invalid argument: %s\n", e.what());
		exit(EXIT_FAILURE);
	}

	printf("Will wait %d seconds before starting...\n", waitTimeSeconds);
	sleep(waitTimeSeconds);

	// Print the vector length to be used, and compute its size
	size_t size = numElements * sizeof(float);
	printf("Vector size: %d elements (%zu bytes)\n", numElements, size);

	float *h_A = (float *)malloc(size);
	float *h_B = (float *)malloc(size);
	float *h_C = (float *)malloc(size);

	if (h_A == NULL || h_B == NULL || h_C == NULL) {
		fprintf(stderr, "Failed to allocate host vectors!\n");
		exit(EXIT_FAILURE);
	}

	// Initialize the host input vectors
	for (int i = 0; i < numElements; ++i) {
		h_A[i] = rand() / (float)RAND_MAX;
		h_B[i] = rand() / (float)RAND_MAX;
	}

	// Allocate the device input vector A
	float *d_A = NULL;
	err = cudaMalloc((void **)&d_A, size);

	if (err != cudaSuccess) {
		fprintf(stderr, "Failed to allocate device vector A (error code %s)!\n",
				cudaGetErrorString(err));
		exit(EXIT_FAILURE);
	}

	// Allocate the device input vector B
	float *d_B = NULL;
	err = cudaMalloc((void **)&d_B, size);

	if (err != cudaSuccess) {
		fprintf(stderr, "Failed to allocate device vector B (error code %s)!\n",
				cudaGetErrorString(err));
		exit(EXIT_FAILURE);
	}

	// Allocate the device output vector C
	float *d_C = NULL;
	err = cudaMalloc((void **)&d_C, size);

	if (err != cudaSuccess) {
		fprintf(stderr, "Failed to allocate device vector C (error code %s)!\n",
				cudaGetErrorString(err));
		exit(EXIT_FAILURE);
	}

	// Copy the host input vectors A and B in host memory to the device input
	// vectors in
	// device memory
	printf("Copy input data from the host memory to the CUDA device\n");
	err = cudaMemcpy(d_A, h_A, size, cudaMemcpyHostToDevice);

	if (err != cudaSuccess) {
		fprintf(
			stderr,
			"Failed to copy vector A from host to device (error code %s)!\n",
			cudaGetErrorString(err));
		exit(EXIT_FAILURE);
	}

	err = cudaMemcpy(d_B, h_B, size, cudaMemcpyHostToDevice);

	if (err != cudaSuccess) {
		fprintf(
			stderr,
			"Failed to copy vector B from host to device (error code %s)!\n",
			cudaGetErrorString(err));
		exit(EXIT_FAILURE);
	}

	// Launch the Vector Add CUDA Kernel
	int threadsPerBlock = 256;
	int blocksPerGrid = (numElements + threadsPerBlock - 1) / threadsPerBlock;
	printf("CUDA kernel launch with %d blocks of %d threads\n", blocksPerGrid,
		   threadsPerBlock);
	vectorSumKernel<<<blocksPerGrid, threadsPerBlock>>>(d_A, d_B, d_C,
														numElements, loops);
	err = cudaGetLastError();

	if (err != cudaSuccess) {
		fprintf(stderr,
				"Failed to launch vectorSumKernel kernel (error code %s)!\n",
				cudaGetErrorString(err));
		exit(EXIT_FAILURE);
	}

	// Copy the device result vector in device memory to the host result vector
	// in host memory.
	printf("Copy output data from the CUDA device to the host memory\n");
	err = cudaMemcpy(h_C, d_C, size, cudaMemcpyDeviceToHost);

	if (err != cudaSuccess) {
		fprintf(
			stderr,
			"Failed to copy vector C from device to host (error code %s)!\n",
			cudaGetErrorString(err));
		exit(EXIT_FAILURE);
	}

	// Verify that the result vector is correct
	for (int i = 0; i < numElements; ++i) {
		if (fabs(h_A[i] + h_B[i] - h_C[i]) > 1e-5) {
			fprintf(stderr, "Result verification failed at element %d!\n", i);
			exit(EXIT_FAILURE);
		}
	}

	printf("Test PASSED\n");

	// Free device global memory
	err = cudaFree(d_A);

	if (err != cudaSuccess) {
		fprintf(stderr, "Failed to free device vector A (error code %s)!\n",
				cudaGetErrorString(err));
		exit(EXIT_FAILURE);
	}

	err = cudaFree(d_B);

	if (err != cudaSuccess) {
		fprintf(stderr, "Failed to free device vector B (error code %s)!\n",
				cudaGetErrorString(err));
		exit(EXIT_FAILURE);
	}

	err = cudaFree(d_C);

	if (err != cudaSuccess) {
		fprintf(stderr, "Failed to free device vector C (error code %s)!\n",
				cudaGetErrorString(err));
		exit(EXIT_FAILURE);
	}

	// Free host memory
	free(h_A);
	free(h_B);
	free(h_C);

	printf("Done\n");
	return 0;
}
