// Package config loads the pinned versions and the image list from
// infrastructure/.
package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"sigs.k8s.io/yaml"
)

// Versions is infrastructure/versions.yaml.
type Versions struct {
	Tools []Tool `json:"tools"`
	Lab   Lab    `json:"lab"`
}

// Tool is one pinned binary.
type Tool struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	URL         string            `json:"url"`
	ArchivePath string            `json:"archivePath,omitempty"`
	SHA256      map[string]string `json:"sha256"`
}

// Lab describes the kind cluster and the registry.
type Lab struct {
	ClusterName      string   `json:"clusterName"`
	NodeImage        string   `json:"nodeImage"`
	KindConfig       string   `json:"kindConfig"`
	Registry         Registry `json:"registry"`
	UnmirroredImages []string `json:"unmirroredImages"`
}

// Registry describes the lab registry container.
type Registry struct {
	Image         string `json:"image"`
	ContainerName string `json:"containerName"`
	VolumeName    string `json:"volumeName"`
	HostPort      int    `json:"hostPort"`
}

// HostAddr is how the host reaches the registry.
func (r Registry) HostAddr() string { return fmt.Sprintf("localhost:%d", r.HostPort) }

// ClusterEndpoint is how cluster nodes reach the registry (Docker network DNS).
func (r Registry) ClusterEndpoint() string { return "http://" + r.ContainerName + ":5000" }

// Images is infrastructure/images.yaml.
type Images struct {
	Images []Image `json:"images"`
}

// Image is one upstream image allowed in the lab.
type Image struct {
	Ref    string   `json:"ref"`
	Digest string   `json:"digest,omitempty"`
	Groups []string `json:"groups"`
	Why    string   `json:"why,omitempty"`
}

var sha256Re = regexp.MustCompile(`^[0-9a-f]{64}$`)

// LoadVersions reads and validates versions.yaml.
func LoadVersions(path string) (*Versions, error) {
	var v Versions
	if err := loadStrict(path, &v); err != nil {
		return nil, err
	}
	for _, t := range v.Tools {
		if t.Name == "" || t.Version == "" || t.URL == "" {
			return nil, fmt.Errorf("%s: tool %q needs name, version and url", path, t.Name)
		}
		if len(t.SHA256) == 0 {
			return nil, fmt.Errorf("%s: tool %q has no sha256 pins", path, t.Name)
		}
		for platform, sum := range t.SHA256 {
			if !sha256Re.MatchString(sum) {
				return nil, fmt.Errorf("%s: tool %q: bad sha256 for %s", path, t.Name, platform)
			}
		}
	}
	l := v.Lab
	if l.ClusterName == "" || l.NodeImage == "" || l.KindConfig == "" || len(l.UnmirroredImages) == 0 ||
		l.Registry.Image == "" || l.Registry.ContainerName == "" || l.Registry.VolumeName == "" || l.Registry.HostPort == 0 {
		return nil, fmt.Errorf("%s: lab section is incomplete", path)
	}
	return &v, nil
}

// Tool returns the pinned tool with the given name.
func (v *Versions) Tool(name string) (Tool, error) {
	for _, t := range v.Tools {
		if t.Name == name {
			return t, nil
		}
	}
	return Tool{}, fmt.Errorf("tool %q is not pinned in versions.yaml", name)
}

// LoadImages reads and validates images.yaml.
func LoadImages(path string) (*Images, error) {
	var im Images
	if err := loadStrict(path, &im); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, i := range im.Images {
		if i.Ref == "" || len(i.Groups) == 0 {
			return nil, fmt.Errorf("%s: image %q needs ref and at least one group", path, i.Ref)
		}
		if strings.Contains(i.Ref, "@") {
			return nil, fmt.Errorf("%s: image %q: put the digest in the digest field, not in ref", path, i.Ref)
		}
		if i.Digest != "" && !sha256Re.MatchString(strings.TrimPrefix(i.Digest, "sha256:")) {
			return nil, fmt.Errorf("%s: image %q: bad digest", path, i.Ref)
		}
		if seen[i.Ref] {
			return nil, fmt.Errorf("%s: image %q listed twice", path, i.Ref)
		}
		seen[i.Ref] = true
	}
	return &im, nil
}

// Group returns the images in the given group; "" or "all" returns everything.
func (im *Images) Group(group string) []Image {
	if group == "" || group == "all" {
		return im.Images
	}
	var out []Image
	for _, i := range im.Images {
		for _, g := range i.Groups {
			if g == group {
				out = append(out, i)
				break
			}
		}
	}
	return out
}

func loadStrict(path string, into any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.UnmarshalStrict(data, into); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
