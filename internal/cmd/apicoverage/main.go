// Command apicoverage checks the retained official Go surface and documented
// capability evidence without compiling cgo packages or contacting the network.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const upstreamRepository = "https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK"
const upstreamCommit = "d51ff1b5baf85dd53155b31543bceba8ec304dc7"
const upstreamVersion = "v2.6.5"

func main() {
	root := flag.String("root", ".", "repository root")
	snapshot := flag.String("snapshot", "", "verified official checkout to inventory instead of checking")
	flag.Parse()
	if err := run(*root, *snapshot); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root, snapshot string) error {
	path := filepath.Join(root, "docs", "api-reference", "official-go.tsv")
	if snapshot != "" {
		return writeSnapshot(snapshot, path)
	}
	reference, err := readReference(path)
	if err != nil {
		return err
	}
	policy, err := readPolicy(filepath.Join(root, "docs", "api-reference", "coverage.json"))
	if err != nil {
		return err
	}
	if err := checkCoverage(root, reference, policy); err != nil {
		return err
	}
	fmt.Printf("API coverage: %d pinned declarations checked; %d explicit compatibility exceptions; %d capability records.\n", len(reference.Symbols), len(policy.Exceptions), len(policy.Capabilities))
	fmt.Println("This checks source contracts and evidence links, not native feature availability or live behavior.")
	return nil
}
