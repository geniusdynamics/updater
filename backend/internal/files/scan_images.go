package files

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type DockerImage struct {
	Registry string
	Repo     string
	Tag      string
	Raw      string
}

func FindDockerImages(dir string, fileNames map[string]bool) ([]DockerImage, error) {
	imageSet := make(map[string]DockerImage)

	registryPattern := `(docker\.io|ghcr\.io|quay\.io|registry\.k8s\.io)`
	imageRegex := regexp.MustCompile(
		registryPattern +
			`/[a-zA-Z0-9._/-]+` +
			`(?::[^\s"]+)?`,
	)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fileName := filepath.Base(path)

		if d.IsDir() {
			return nil
		}

		if _, exists := fileNames[fileName]; !exists {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		content := stripComments(string(data))
		vars := extractBashVars(content)

		matches := imageRegex.FindAllString(content, -1)
		for _, raw := range matches {
			resolved := resolveVars(raw, vars)
			img := parseImage(resolved)
			imageSet[img.Raw] = img
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	images := make([]DockerImage, 0, len(imageSet))
	for _, img := range imageSet {
		images = append(images, img)
	}

	return images, nil
}

func parseImage(raw string) DockerImage {
	tag := "latest"

	// split registry / rest
	parts := strings.SplitN(raw, "/", 2)
	registry := parts[0]
	repoAndTag := parts[1]

	if strings.Contains(repoAndTag, ":") {
		rt := strings.SplitN(repoAndTag, ":", 2)
		repoAndTag = rt[0]
		tag = rt[1]
	}

	return DockerImage{
		Registry: registry,
		Repo:     repoAndTag,
		Tag:      tag,
		Raw:      raw,
	}
}

func resolveVars(input string, vars map[string]string) string {
	varRef := regexp.MustCompile(`\$\{([A-Za-z0-9_]+)\}`)

	return varRef.ReplaceAllStringFunc(input, func(m string) string {
		name := varRef.FindStringSubmatch(m)[1]
		if v, ok := vars[name]; ok {
			return v
		}
		return m // leave untouched if unknown
	})
}

func extractBashVars(content string) map[string]string {
	vars := make(map[string]string)

	// Matches: name="value" or name=value
	varRegex := regexp.MustCompile(
		`(?m)^([A-Za-z_][A-Za-z0-9_]*)=(?:"([^"]+)"|([^\s#]+))`,
	)

	matches := varRegex.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		key := m[1]
		val := m[2]
		if val == "" {
			val = m[3]
		}
		vars[key] = val
	}

	return vars
}

func stripComments(content string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		var cleaned strings.Builder
		inSingle := false
		inDouble := false
		for i := 0; i < len(line); i++ {
			ch := line[i]

			switch ch {
			case '\'':
				if !inDouble {
					inSingle = !inSingle
				}
			case '"':
				if !inSingle {
					inDouble = !inDouble
				}
			case '#':
				if !inSingle && !inDouble {
					// stop processing the line
					i = len(line)
					continue
				}
			}

			if i < len(line) {
				cleaned.WriteByte(ch)
			}
		}

		result = append(result, cleaned.String())
	}
	return strings.Join(result, "\n")
}
