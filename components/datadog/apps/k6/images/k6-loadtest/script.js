import http from 'k6/http';
import { check } from 'k6';
import { Trend } from 'k6/metrics';

// A custom metric to track the download time of the S3 blob.
const downloadTime = new Trend('download_time');

export const options = {
    stages: [
        // Use a single virtual user for a long duration to continuously download.
        { duration: '15m', target: 1 },
    ],
};

export default function () {
    const url = 'https://containerimageregistry.s3.us-east-1.amazonaws.com/blobs/sha256%3A41db880336816e5c5e72399bb31af15413c4971adc89c624786ea3d236dad0f9';

    // Make the GET request to download the blob
    const res = http.get(url);

    // Record the download time only if the request was successful and timing data is available.
    if (res && res.status === 200 && res.timing) {
        downloadTime.add(res.timing.duration);
    }

    // Check is still useful for reporting the overall success rate.
    check(res, {
        'status is 200': (r) => r.status === 200,
    });

    // No sleep, to download as frequently as possible.
} 