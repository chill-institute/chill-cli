package main

type packageJSON struct {
	Name                 string            `json:"name"`
	Version              string            `json:"version"`
	Description          string            `json:"description"`
	License              string            `json:"license"`
	Repository           repository        `json:"repository"`
	Bugs                 bugs              `json:"bugs"`
	Homepage             string            `json:"homepage"`
	Keywords             []string          `json:"keywords,omitempty"`
	Bin                  map[string]string `json:"bin,omitempty"`
	Files                []string          `json:"files"`
	OS                   []string          `json:"os,omitempty"`
	CPU                  []string          `json:"cpu,omitempty"`
	OptionalDependencies map[string]string `json:"optionalDependencies,omitempty"`
	PublishConfig        publishConfig     `json:"publishConfig"`
}

type repository struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type bugs struct {
	URL string `json:"url"`
}

type publishConfig struct {
	Access string `json:"access"`
}

func rootPackage(version string) packageJSON {
	optionalDependencies := map[string]string{}
	for _, t := range targets {
		optionalDependencies[platformPackageName(t)] = version
	}

	pkg := basePackage(rootPackageName, version, "Search chill.institute and send transfers from the terminal")
	pkg.Bin = map[string]string{binaryName: "bin/chilly.js"}
	pkg.Files = []string{"bin"}
	pkg.Keywords = []string{"chill.institute", "cli", "chilly"}
	pkg.OptionalDependencies = optionalDependencies
	return pkg
}

func platformPackage(version string, t target) packageJSON {
	pkg := basePackage(platformPackageName(t), version, "chilly binary for "+t.npmOS+" "+t.npmArch)
	pkg.Files = []string{"bin"}
	pkg.OS = []string{t.npmOS}
	pkg.CPU = []string{t.npmArch}
	return pkg
}

func basePackage(name string, version string, description string) packageJSON {
	return packageJSON{
		Name:        name,
		Version:     version,
		Description: description,
		License:     "MIT",
		Repository: repository{
			Type: "git",
			URL:  repositoryURL,
		},
		Bugs: bugs{
			URL: "https://github.com/chill-institute/chill-cli/issues",
		},
		Homepage: homepageURL,
		PublishConfig: publishConfig{
			Access: "public",
		},
	}
}
