package images

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

// Tag represents a single image tag with optional semantic version
type Tag struct {
	Name    string `json:"name"`              // Raw tag name
	Version string `json:"version,omitempty"` // Parsed semantic version, if available
}

// DockerHubTagsResponse represents Docker Hub API response
type DockerHubTagsResponse struct {
	Results []struct {
		Name string `json:"name"`
	} `json:"results"`
	Next string `json:"next"`
}

// GenericTagsResponse represents GHCR, Quay, K8s style response
type GenericTagsResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// Regex to parse semantic versions like v1.2.3
var semverRegex = regexp.MustCompile(`v?(\d+\.\d+\.\d+)`)

// parseVersion extracts a semantic version from a tag string
func parseVersion(tag string) string {
	match := semverRegex.FindStringSubmatch(tag)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

// baseURLGenerator returns the API endpoint for a registry/repo
func baseURLGenerator(registry, repo string) string {
	switch registry {
	case "docker.io":
		return fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/tags?page_size=100", repo)
	case "ghcr.io":
		return fmt.Sprintf("https://ghcr.io/v2/%s/tags/list", repo)
	case "quay.io":
		return fmt.Sprintf("https://quay.io/v2/%s/tags/list", repo)
	case "registry.k8s.io":
		return fmt.Sprintf("https://registry.k8s.io/v2/%s/tags/list", repo)
	default:
		fmt.Printf("registry unsupported")
		return ""
	}
}

func FindNearestUpgrade(current string, tags []Tag) *Tag {
	currMaj, currMin, currPat, ok := parseSemver(current)
	if !ok {
		return nil
	}

	var best *Tag

	for _, t := range tags {
		maj, min, pat, ok := parseSemver(t.Version)
		if !ok {
			continue
		}

		if greater(maj, min, pat, currMaj, currMin, currPat) {
			if best == nil {
				tmp := t
				best = &tmp
				continue
			}

			bMaj, bMin, bPat, _ := parseSemver(best.Version)

			// keep the smallest version that is still greater
			if greater(bMaj, bMin, bPat, maj, min, pat) {
				tmp := t
				best = &tmp
			}
		}
	}

	return best
}

// GetImageUpdates fetches tags for a given registry and repo
func GetImageUpdates(registry, repo string) ([]Tag, error) {
	baseURL := baseURLGenerator(registry, repo)
	if baseURL == "" {
		return nil, fmt.Errorf("unsupported registry: %s", registry)
	}
	var (
		tags []Tag
		err  error
	)

	switch registry {
	case "docker.io":
		tags, err = getDockerHubTags(baseURL)
	default:
		tags, err = getGenericTags(baseURL)
	}

	return filterLatestVersion(tags), err
}

// getDockerHubTags handles Docker Hub API with pagination
func getDockerHubTags(url string) ([]Tag, error) {
	tags := []Tag{}
	client := &http.Client{}

	for url != "" {
		resp, err := client.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var dockerResp DockerHubTagsResponse
		if err := json.Unmarshal(body, &dockerResp); err != nil {
			return nil, err
		}

		for _, r := range dockerResp.Results {
			tags = append(tags, Tag{
				Name:    r.Name,
				Version: parseVersion(r.Name),
			})
		}

		url = dockerResp.Next // continue to next page
	}

	return tags, nil
}

// getGenericTags handles GHCR, Quay, K8s style APIs
func getGenericTags(url string) ([]Tag, error) {
	client := &http.Client{}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var genericResp GenericTagsResponse
	if err := json.Unmarshal(body, &genericResp); err != nil {
		return nil, err
	}

	tags := []Tag{}
	for _, t := range genericResp.Tags {
		tags = append(tags, Tag{
			Name:    t,
			Version: parseVersion(t),
		})
	}

	return tags, nil
}

func filterLatestVersion(tags []Tag) []Tag {
	versionMap := map[string]Tag{}

	for _, t := range tags {
		if t.Version == "" {
			continue // skip tags without semantic version
		}
		// Remove architecture prefixes if any (like arm64-)
		versionParts := strings.Split(t.Version, "-")
		v := versionParts[len(versionParts)-1]

		versionMap[v] = t
	}

	if len(versionMap) == 0 {
		return nil
	}

	// Find the highest version
	versions := []string{}
	for v := range versionMap {
		versions = append(versions, v)
	}

	// Sort using simple semantic version sort
	sort.Slice(versions, func(i, j int) bool {
		return compareSemver(versions[i], versions[j]) > 0 // descending
	})

	latestVersion := versions[0]
	return []Tag{versionMap[latestVersion]}
}

// compareSemver compares two semantic versions, returns 1 if v1>v2, -1 if v1<v2, 0 if equal
func compareSemver(v1, v2 string) int {
	var major1, minor1, patch1 int
	var major2, minor2, patch2 int

	fmt.Sscanf(v1, "%d.%d.%d", &major1, &minor1, &patch1)
	fmt.Sscanf(v2, "%d.%d.%d", &major2, &minor2, &patch2)

	if major1 != major2 {
		if major1 > major2 {
			return 1
		}
		return -1
	}
	if minor1 != minor2 {
		if minor1 > minor2 {
			return 1
		}
		return -1
	}
	if patch1 != patch2 {
		if patch1 > patch2 {
			return 1
		}
		return -1
	}
	return 0
}

func newTag(name string) Tag {
	return Tag{
		Name:    name,
		Version: extractVersion(name),
	}
}

func extractVersion(tag string) string {
	match := semverRegex.FindStringSubmatch(tag)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func parseSemver(v string) (int, int, int, bool) {
	var maj, min, pat int
	_, err := fmt.Sscanf(v, "%d.%d.%d", &maj, &min, &pat)
	if err != nil {
		return 0, 0, 0, false
	}
	return maj, min, pat, true
}

func greater(aMaj, aMin, aPat, bMaj, bMin, bPat int) bool {
	if aMaj != bMaj {
		return aMaj > bMaj
	}
	if aMin != bMin {
		return aMin > bMin
	}
	return aPat > bPat
}
