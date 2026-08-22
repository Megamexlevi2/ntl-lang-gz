package meta

import (
	"encoding/json"
	"fmt"
	"sync"
)

var (
	_versionData []byte
	_once        sync.Once
	_cached      versionInfo
)

func SetVersionData(data []byte) {
	_versionData = data
	_once = sync.Once{}
}

type versionInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	GitHub      string   `json:"github"`
	Repository  string   `json:"repository"`
	License     string   `json:"license"`
	Year        string   `json:"year"`
	BuildDate   string   `json:"buildDate"`
	Description string   `json:"description"`
	Languages   []string `json:"languages"`
	MinGo       string   `json:"minGo"`
	Platforms   []string `json:"platforms"`
}

func loadVersion() versionInfo {
	_once.Do(func() {
		_ = json.Unmarshal(_versionData, &_cached)
	})
	return _cached
}

func Version() string {
	v := loadVersion()
	if v.Version == "" {
		return "unknown"
	}
	return v.Version
}

func Name() string {
	v := loadVersion()
	if v.Name == "" {
		return "Lunex"
	}
	return v.Name
}

func Author() string {
	return loadVersion().Author
}

func Year() string {
	return loadVersion().Year
}

func BuildDate() string {
	return loadVersion().BuildDate
}

func Repository() string {
	return loadVersion().Repository
}

func License() string {
	return loadVersion().License
}

func Description() string {
	return loadVersion().Description
}

func GitHub() string {
	return loadVersion().GitHub
}

func FullVersion() string { return Version() }

func PrintVersion() {
	v := loadVersion()
	name := v.Name
	if name == "" {
		name = "Lunex"
	}
	fmt.Printf("%s %s\n", name, v.Version)
	if v.BuildDate != "" {
		fmt.Printf("build    %s\n", v.BuildDate)
	}
	if v.Author != "" {
		fmt.Printf("author   %s\n", v.Author)
	}
	if v.License != "" {
		fmt.Printf("license  %s\n", v.License)
	}
	if v.Repository != "" {
		fmt.Printf("repo     %s\n", v.Repository)
	}
}
