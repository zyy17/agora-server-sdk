package main

import (
	"bufio"
	"crypto/sha1" // Git object IDs are provenance checks, not security signatures.
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type reference struct {
	Symbols map[string]symbol
	Sources map[string]string
}

type sourceFile struct {
	Path string `json:"path"`
	SHA  string `json:"sha"`
}

func writeSnapshot(upstream, destination string) error {
	sources, err := verifiedSources(upstream)
	if err != nil {
		return err
	}
	var output strings.Builder
	fmt.Fprintf(&output, "# official-go-api-v1\n# repository\t%s\n# version\t%s\n# commit\t%s\n", upstreamRepository, upstreamVersion, upstreamCommit)
	output.WriteString("# Scope: exported Go declarations, struct fields and interface methods; all build variants.\n# Excludes unexported C bridges, test files, implementation bodies and native-only APIs.\n# Columns: package, kind, name, canonical signature, official source file.\n")
	for _, file := range sources {
		fmt.Fprintf(&output, "# source\t%s\t%s\n", file.Path, file.SHA)
	}
	for _, pkg := range []string{"rtc", "rtm"} {
		items, err := surface(upstream, "go_sdk/"+pkg, false)
		if err != nil {
			return err
		}
		for _, item := range sortedSymbols(items) {
			fmt.Fprintf(&output, "%s\t%s\t%s\t%s\t%s\n", "agora/"+pkg, item.Kind, item.Name, item.Signature, item.File)
		}
	}
	return os.WriteFile(destination, []byte(output.String()), 0o644)
}

func verifiedSources(root string) ([]sourceFile, error) {
	command := exec.Command("git", "-C", root, "rev-parse", "HEAD")
	commit, gitErr := command.Output()
	var files []sourceFile
	if gitErr == nil && strings.TrimSpace(string(commit)) == upstreamCommit {
		output, err := exec.Command("git", "-C", root, "ls-tree", "-r", upstreamCommit, "--", "go_sdk/rtc", "go_sdk/rtm").Output()
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 4 && publicSource(fields[3]) {
				files = append(files, sourceFile{fields[3], fields[2]})
			}
		}
	} else {
		// Connector-materialized trees carry the same per-file Git object IDs.
		data, err := os.ReadFile(filepath.Join(root, "source-manifest.json"))
		if err != nil {
			return nil, fmt.Errorf("snapshot requires clean pinned git checkout or verified source-manifest.json: %w", err)
		}
		var manifest struct {
			Repository string       `json:"repository"`
			Ref        string       `json:"ref"`
			Tag        string       `json:"tag"`
			RTC        []sourceFile `json:"rtc"`
			RTM        []sourceFile `json:"rtm"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, err
		}
		if manifest.Repository != strings.TrimPrefix(upstreamRepository, "https://github.com/") || manifest.Ref != upstreamCommit || manifest.Tag != upstreamVersion {
			return nil, fmt.Errorf("source manifest provenance does not match pinned official release")
		}
		files = append(manifest.RTC, manifest.RTM...)
	}
	seen := make(map[string]bool)
	for _, file := range files {
		if !publicSource(file.Path) || seen[file.Path] {
			return nil, fmt.Errorf("invalid or duplicate source path %q", file.Path)
		}
		seen[file.Path] = true
		data, err := os.ReadFile(filepath.Join(root, file.Path))
		if err != nil {
			return nil, err
		}
		hash := sha1.New()
		fmt.Fprintf(hash, "blob %d%c", len(data), 0)
		hash.Write(data)
		if fmt.Sprintf("%x", hash.Sum(nil)) != file.SHA {
			return nil, fmt.Errorf("official Git blob mismatch for %s", file.Path)
		}
	}
	for _, pkg := range []string{"rtc", "rtm"} {
		entries, err := os.ReadDir(filepath.Join(root, "go_sdk", pkg))
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			path := "go_sdk/" + pkg + "/" + entry.Name()
			if publicSource(path) && !seen[path] {
				return nil, fmt.Errorf("source has no pinned provenance: %s", path)
			}
		}
	}
	if len(files) != 40 {
		return nil, fmt.Errorf("expected 40 official Go source files, got %d", len(files))
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func publicSource(path string) bool {
	return (strings.HasPrefix(path, "go_sdk/rtc/") || strings.HasPrefix(path, "go_sdk/rtm/")) &&
		strings.Count(path, "/") == 2 && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}

func readReference(path string) (reference, error) {
	result := reference{Symbols: make(map[string]symbol), Sources: make(map[string]string)}
	file, err := os.Open(path)
	if err != nil {
		return result, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var repository, version, commit string
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")
		if strings.HasPrefix(line, "#") {
			if len(fields) == 2 {
				switch fields[0] {
				case "# repository":
					repository = fields[1]
				case "# version":
					version = fields[1]
				case "# commit":
					commit = fields[1]
				}
			}
			if len(fields) == 3 && fields[0] == "# source" {
				result.Sources[fields[1]] = fields[2]
			}
			continue
		}
		if len(fields) != 5 {
			return result, fmt.Errorf("invalid reference row: %q", line)
		}
		item := symbol{fields[0], fields[1], fields[2], fields[3], fields[4]}
		if _, exists := result.Symbols[item.key()]; exists {
			return result, fmt.Errorf("duplicate reference %s", item.key())
		}
		if len(result.Sources[item.File]) != 40 {
			return result, fmt.Errorf("missing source provenance for %s", item.key())
		}
		result.Symbols[item.key()] = item
	}
	if err := scanner.Err(); err != nil {
		return result, err
	}
	if repository != upstreamRepository || version != upstreamVersion || commit != upstreamCommit || len(result.Sources) != 40 || len(result.Symbols) != 1159 {
		return result, fmt.Errorf("reference provenance is incomplete or does not match pinned release")
	}
	return result, nil
}
