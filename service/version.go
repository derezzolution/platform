package service

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Version struct {
	// Build is the primary service build information.
	Build Build `json:"build"`

	// BuildTime is the time that the build happened (ideally aligns with the package timestamp),
	// e.g.: $(date -u +"%Y-%m-%dT%H:%M:%SZ") -> "2023-02-26T07:49:35Z" (UTC)
	BuildTime time.Time `json:"buildTime"`

	// Pipeline provides the build method, e.g.: "manual", "github_actions", "bitbucket_pipelines"
	Pipeline string `json:"pipeline"`

	// (Optional) If supplied, provides the build information for the resource bundle.
	Resources Build `json:"resources"`
}

// NewVersion creates a new version given a package file system (which must contain version.json).
func NewVersion(packageFS *embed.FS) (*Version, error) {
	if packageFS == nil {
		return &Version{}, fmt.Errorf("packageFS is not defined")
	}

	// Open the packaged version file
	versionJson, err := packageFS.ReadFile("version.json")
	if err != nil {
		return &Version{}, err
	}

	// Decode the version information
	decoder := json.NewDecoder(strings.NewReader(string(versionJson)))
	ret := &Version{}
	err = decoder.Decode(ret)
	if err != nil {
		return &Version{}, err
	}

	return ret, nil
}

func (v *Version) VersionHash() string {
	return v.Build.BuildHash()
}

func (v *Version) LogSummary() {
	log.Printf("%s Build %s", filepath.Base(os.Args[0]), v.VersionHash())
}

func (v *Version) ToJson() (string, error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

type Build struct {
	BuildNumber int    `json:"buildNumber"`
	Hash        string `json:"hash"`
	ShortHash   string `json:"shortHash"`
}

func (b *Build) BuildHash() string {
	return fmt.Sprintf("%d-%s", b.BuildNumber, b.ShortHash)
}
