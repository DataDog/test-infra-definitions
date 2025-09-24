package kubernetes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver"
)

// KindConfig contains the kind version and the kind node image to use
type KindConfig struct {
	KindVersion       string
	NodeImageVersion  string
	KubeVersion       string // Clean Kubernetes version for semantic parsing
	UsePublicRegistry bool   // If true, pull from docker.io instead of internal mirror
}

// DockerHubTag represents a tag from Docker Hub API
type DockerHubTag struct {
	Name      string `json:"name"`
	Digest    string `json:"digest"`
	FullSize  int64  `json:"full_size"`
	TagStatus string `json:"tag_status"`
}

// DockerHubResponse represents the response from Docker Hub API
type DockerHubResponse struct {
	Results []DockerHubTag `json:"results"`
	Next    string         `json:"next"`
}

// Source: https://github.com/kubernetes-sigs/kind/releases
var kubeToKindVersion = map[string]KindConfig{
	"1.32": {
		KindVersion:      "v0.26.0",
		NodeImageVersion: "v1.32.0@sha256:c48c62eac5da28cdadcf560d1d8616cfa6783b58f0d94cf63ad1bf49600cb027",
	},
	"1.31": {
		KindVersion:      "v0.26.0",
		NodeImageVersion: "v1.31.4@sha256:2cb39f7295fe7eafee0842b1052a599a4fb0f8bcf3f83d96c7f4864c357c6c30",
	},
	"1.30": {
		KindVersion:      "v0.26.0",
		NodeImageVersion: "v1.30.8@sha256:17cd608b3971338d9180b00776cb766c50d0a0b6b904ab4ff52fd3fc5c6369bf",
	},
	"1.29": {
		KindVersion:      "v0.22.0",
		NodeImageVersion: "v1.29.2@sha256:51a1434a5397193442f0be2a297b488b6c919ce8a3931be0ce822606ea5ca245",
	},
	"1.28": {
		KindVersion:      "v0.22.0",
		NodeImageVersion: "v1.28.7@sha256:9bc6c451a289cf96ad0bbaf33d416901de6fd632415b076ab05f5fa7e4f65c58",
	},
	"1.27": {
		KindVersion:      "v0.22.0",
		NodeImageVersion: "v1.27.11@sha256:681253009e68069b8e01aad36a1e0fa8cf18bb0ab3e5c4069b2e65cafdd70843",
	},
	"1.26": {
		KindVersion:      "v0.22.0",
		NodeImageVersion: "v1.26.14@sha256:5d548739ddef37b9318c70cb977f57bf3e5015e4552be4e27e57280a8cbb8e4f",
	},
	"1.25": {
		KindVersion:      "v0.22.0",
		NodeImageVersion: "v1.25.16@sha256:e8b50f8e06b44bb65a93678a65a26248fae585b3d3c2a669e5ca6c90c69dc519",
	},
	"1.24": {
		KindVersion:      "v0.22.0",
		NodeImageVersion: "v1.24.17@sha256:bad10f9b98d54586cba05a7eaa1b61c6b90bfc4ee174fdc43a7b75ca75c95e51",
	},
	"1.23": {
		KindVersion:      "v0.22.0",
		NodeImageVersion: "v1.23.17@sha256:14d0a9a892b943866d7e6be119a06871291c517d279aedb816a4b4bc0ec0a5b3",
	},
	"1.22": {
		KindVersion:      "v0.20.0",
		NodeImageVersion: "v1.22.17@sha256:f5b2e5698c6c9d6d0adc419c0deae21a425c07d81bbf3b6a6834042f25d4fba2",
	},
	"1.21": {
		KindVersion:      "v0.20.0",
		NodeImageVersion: "v1.21.14@sha256:8a4e9bb3f415d2bb81629ce33ef9c76ba514c14d707f9797a01e3216376ba093",
	},
	"1.20": {
		KindVersion:      "v0.17.0",
		NodeImageVersion: "v1.20.15@sha256:a32bf55309294120616886b5338f95dd98a2f7231519c7dedcec32ba29699394",
	},
	"1.19": {
		KindVersion:      "v0.17.0",
		NodeImageVersion: "v1.19.16@sha256:476cb3269232888437b61deca013832fee41f9f074f9bed79f57e4280f7c48b7",
	},
	// Use ubuntu 20.04 for the below k8s versions
	"1.18": {
		KindVersion:      "v0.17.0",
		NodeImageVersion: "v1.18.20@sha256:61c9e1698c1cb19c3b1d8151a9135b379657aee23c59bde4a8d87923fcb43a91",
	},
	"1.17": {
		KindVersion:      "v0.17.0",
		NodeImageVersion: "v1.17.17@sha256:e477ee64df5731aa4ef4deabbafc34e8d9a686b49178f726563598344a3898d5",
	},
	"1.16": {
		KindVersion:      "v0.15.0",
		NodeImageVersion: "v1.16.15@sha256:64bac16b83b6adfd04ea3fbcf6c9b5b893277120f2b2cbf9f5fa3e5d4c2260cc",
	},
}

// getKindVersionForKubernetes determines the appropriate Kind version for a given Kubernetes version
// Based on Kind release compatibility: https://github.com/kubernetes-sigs/kind/releases
// Used as fallback if dynamic resolution fails
func getKindVersionForKubernetes(kubeVersion *semver.Version) string {
	major := kubeVersion.Major()
	minor := kubeVersion.Minor()

	// For Kubernetes 1.34+, use Kind v0.30.0+
	if major == 1 && minor >= 34 {
		return "v0.30.0"
	}

	// For older versions, use Kind v0.26.0
	if major == 1 && minor >= 30 {
		return "v0.26.0"
	}

	// For very old versions, use an older Kind version
	return "v0.22.0"
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	PreRelease bool   `json:"prerelease"`
}

// getLatestKindVersionDynamic fetches the latest Kind version from GitHub releases API
func getLatestKindVersionDynamic() (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	// Fetch releases from GitHub API
	githubURL := "https://api.github.com/repos/kubernetes-sigs/kind/releases"

	resp, err := client.Get(githubURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch GitHub releases: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gitHub API returned status %d", resp.StatusCode)
	}

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("failed to decode GitHub response: %v", err)
	}

	// Find the latest non-draft, non-prerelease version
	versionRegex := regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
	var versions []*semver.Version

	for _, release := range releases {
		if release.Draft || release.PreRelease {
			continue
		}

		if versionRegex.MatchString(release.TagName) {
			if version, err := semver.NewVersion(release.TagName); err == nil {
				versions = append(versions, version)
			}
		}
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no valid Kind versions found in GitHub releases")
	}

	sort.Sort(sort.Reverse(semver.Collection(versions)))
	latestVersion := versions[0]

	fmt.Printf("Found %d valid Kind versions, latest is: %s\n", len(versions), latestVersion.String())
	return "v" + latestVersion.String(), nil
}

// getLatestKindVersion fetches the latest Kubernetes version from Docker Hub
func getLatestKindVersionConfig() (*KindConfig, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	// Fetch tags from Docker Hub API
	dockerHubURL := "https://hub.docker.com/v2/repositories/kindest/node/tags?page_size=100"
	resp, err := client.Get(dockerHubURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Docker Hub tags: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker Hub API returned status %d", resp.StatusCode)
	}

	var dockerResp DockerHubResponse
	if err := json.NewDecoder(resp.Body).Decode(&dockerResp); err != nil {
		return nil, fmt.Errorf("failed to decode Docker Hub response: %v", err)
	}

	// Filter and sort versions - look for active tags
	kubeVersionRegex := regexp.MustCompile(`^v(\d+\.\d+\.\d+)$`)
	var versions []*semver.Version
	tagToDigest := make(map[string]string)

	for _, tag := range dockerResp.Results {
		// Only process active tags
		if tag.TagStatus != "active" {
			continue
		}

		matches := kubeVersionRegex.FindStringSubmatch(tag.Name)
		if len(matches) >= 2 {
			if version, err := semver.NewVersion(matches[1]); err == nil {
				versions = append(versions, version)
				// Create full tag with digest (format: v1.33.2@sha256:...)
				fullTag := fmt.Sprintf("%s@%s", tag.Name, tag.Digest)
				tagToDigest[version.String()] = fullTag
			}
		}
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no valid active Kubernetes versions found in Docker Hub")
	}

	sort.Sort(sort.Reverse(semver.Collection(versions)))
	latestVersion := versions[0]
	fullTag := tagToDigest[latestVersion.String()]

	// Attempt to use latest kind version
	kindVersion, err := getLatestKindVersionDynamic()
	if err != nil {
		kindVersion = getKindVersionForKubernetes(latestVersion)
	}
	fmt.Printf("Selected Kind version %s for Kubernetes %s\n", kindVersion, latestVersion.String())

	return &KindConfig{
		KindVersion:       kindVersion,
		NodeImageVersion:  fullTag,
		KubeVersion:       latestVersion.String(), // Clean version for semantic parsing
		UsePublicRegistry: true,                   // Latest versions must be pulled from Docker Hub
	}, nil
}

// GetKindVersionConfig returns the kind version and the kind node image to use based on kubernetes version
func GetKindVersionConfig(kubeVersion string) (*KindConfig, error) {
	// Handle "latest" as a special case
	if kubeVersion == "latest" {
		return getLatestKindVersionConfig()
	}

	kubeSemVer, err := semver.NewVersion(kubeVersion)
	if err != nil {
		return nil, err
	}

	kindVersionConfig, found := kubeToKindVersion[fmt.Sprintf("%d.%d", kubeSemVer.Major(), kubeSemVer.Minor())]
	if !found {
		return nil, fmt.Errorf("unsupported kubernetes version. Supported versions are %s", strings.Join(kubeSupportedVersions(), ", "))
	}

	// Ensure KubeVersion is populated for static configs too
	kindVersionConfig.KubeVersion = kubeVersion
	return &kindVersionConfig, nil
}

// kubeSupportedVersions returns a comma-separated list of supported kubernetes versions
func kubeSupportedVersions() []string {
	versions := make([]string, 0, len(kubeToKindVersion))

	for kubeVersion := range kubeToKindVersion {
		versions = append(versions, kubeVersion)
	}

	return versions
}
