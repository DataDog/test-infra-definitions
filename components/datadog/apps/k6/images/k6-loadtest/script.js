import http from 'k6/http';
import { check } from 'k6';
import { Trend } from 'k6/metrics';

// A custom metric to track the total time of the registry pull flow.
const downloadTime = new Trend('download_time');
// A custom metric to track time spent following redirects (e.g., registry -> S3).
const redirectTime = new Trend('redirect_time');

export const options = {
    stages: [
        // Use a single virtual user for a long duration to continuously download.
        { duration: '15m', target: 1 },
    ],
};

// Emulate a full Docker Registry v2 image pull against a target registry.
// Sequence:
// 1) GET  /v2/
// 2) GET  /v2/<repo>/manifests/<tag>                    (manifest list)
// 3) GET  /v2/<repo>/manifests/<mono-arch-manifest-digest>
// 4) GET  /v2/<repo>/blobs/<config-digest>
// 5) GET  /v2/<repo>/blobs/<layer-digest> (one or more)

const REGISTRY_HOST = __ENV.REGISTRY_HOST || 'adel-reg.com';
const REPOSITORY = __ENV.REPOSITORY || 'agent';
const IMAGE_TAG = __ENV.IMAGE_TAG || '7.70.0';
const TARGET_ARCH = __ENV.TARGET_ARCH || 'amd64';
const TARGET_OS = __ENV.TARGET_OS || 'linux';

function getJSON(url, headers = {}) {
    const res = http.get(url, { headers });
    let body = null;
    if (res && res.status === 200) {
        try {
            body = res.json();
        } catch (e) {
            // ignore JSON parse errors; checks below will fail and be reported
        }
    }
    return { res, body };
}

function resolveMonoArchDigestFromList(listJson) {
    if (!listJson || !listJson.manifests) {
        return null;
    }
    for (const entry of listJson.manifests) {
        const platform = entry.platform || {};
        if (
            platform.os === TARGET_OS &&
            (platform.architecture === TARGET_ARCH || (platform.architecture === 'arm64/v8' && TARGET_ARCH === 'arm64'))
        ) {
            return entry.digest;
        }
    }
    return null;
}

export default function () {
    let totalDurationMs = 0;

    // 1) Ping the registry base endpoint
    const baseRes = http.get(`https://${REGISTRY_HOST}/v2/`);
    totalDurationMs += baseRes && baseRes.timings ? baseRes.timings.duration : 0;
    check(baseRes, { 'GET /v2 returns 200': (r) => r.status === 200 });

    // 2) Fetch manifest list for the tag
    const manifestListUrl = `https://${REGISTRY_HOST}/v2/${REPOSITORY}/manifests/${IMAGE_TAG}`;
    const acceptManifestList = {
        Accept:
            'application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.index.v1+json',
    };
    const { res: mlRes, body: mlJson } = getJSON(manifestListUrl, acceptManifestList);
    totalDurationMs += mlRes && mlRes.timings ? mlRes.timings.duration : 0;
    check(mlRes, { 'manifest list 200': (r) => r.status === 200 });

    const monoArchDigest = resolveMonoArchDigestFromList(mlJson);
    check(monoArchDigest, { 'mono-arch digest resolved': (d) => typeof d === 'string' && d.startsWith('sha256:') });

    // 3) Fetch specific image manifest by digest
    const manifestUrl = `https://${REGISTRY_HOST}/v2/${REPOSITORY}/manifests/${monoArchDigest}`;
    const acceptManifest = {
        Accept: 'application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json',
    };
    const { res: mRes, body: mJson } = getJSON(manifestUrl, acceptManifest);
    totalDurationMs += mRes && mRes.timings ? mRes.timings.duration : 0;
    check(mRes, { 'manifest 200': (r) => r.status === 200 });

    const configDigest = mJson && mJson.config && mJson.config.digest;
    check(configDigest, { 'config digest present': (d) => typeof d === 'string' && d.startsWith('sha256:') });

    const layerDigests = (mJson && mJson.layers ? mJson.layers.map((l) => l.digest) : []).filter(
        (d) => typeof d === 'string' && d.startsWith('sha256:')
    );
    check(layerDigests, { 'at least one layer digest': (arr) => arr.length > 0 });

    // 4) Fetch config blob (measure redirect time if any)
    const configUrl = `https://${REGISTRY_HOST}/v2/${REPOSITORY}/blobs/${configDigest}`;
    const cfg = getFollowingRedirect(configUrl);
    totalDurationMs += cfg.durationMs;
    check(cfg.res, { 'config blob OK': (r) => r.status >= 200 && r.status < 300 });

    // 5) Fetch one or two layer blobs to mimic pull data path
    const layersToFetch = Math.min(layerDigests.length, 2);
    for (let i = 0; i < layersToFetch; i++) {
        const layerUrl = `https://${REGISTRY_HOST}/v2/${REPOSITORY}/blobs/${layerDigests[i]}`;
        const layer = getFollowingRedirect(layerUrl);
        totalDurationMs += layer.durationMs;
        check(layer.res, { [`layer ${i} blob OK`]: (r) => r.status >= 200 && r.status < 300 });
    }

    // Record total duration of the full flow
    downloadTime.add(totalDurationMs);
}

// Fetch a URL without auto-following redirects, measuring redirect steps and final duration.
function getFollowingRedirect(url) {
    let current = url;
    let attempts = 0;
    let durationMs = 0;
    const maxRedirects = 5;
    while (attempts < maxRedirects) {
        const r = http.get(current, { redirects: 0 });
        if (r && r.timings) {
            durationMs += r.timings.duration;
        }
        // 3xx redirect with Location header
        if (r && r.status >= 300 && r.status < 400 && r.headers && r.headers.Location) {
            if (r.timings) {
                redirectTime.add(r.timings.duration);
            }
            current = r.headers.Location;
            attempts += 1;
            continue;
        }
        // Return the last response (ideally 2xx)
        return { res: r, durationMs };
    }
    // if we exhausted redirects, return the last seen response
    const fallback = http.get(current, { redirects: 0 });
    if (fallback && fallback.timings) {
        durationMs += fallback.timings.duration;
    }
    return { res: fallback, durationMs };
}