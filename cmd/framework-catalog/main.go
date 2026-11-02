// Framework catalog source verification is an operator-only offline tool.
// It does not publish content, grant access or mutate tenant Programs.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/CloudSpaceLab/clearsight-grc/internal/frameworkcatalog"
)

type verificationReceipt struct {
	PublisherCode string `json:"publisher_code"`
	FrameworkCode string `json:"framework_code"`
	EditionCode   string `json:"edition_code"`
	ClauseCount   int    `json:"clause_count"`
	SourceBytes   int64  `json:"source_bytes"`
	SourceSHA256  string `json:"source_sha256"`
	ReleaseSHA256 string `json:"release_sha256"`
	Published     bool   `json:"published"`
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("framework-catalog", flag.ContinueOnError)
	flags.SetOutput(stderr)
	manifestPath := flags.String("manifest", "", "Reviewed JSON edition manifest")
	sourcePath := flags.String("source", "", "Local authoritative source file matching manifest hash")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *manifestPath == "" || *sourcePath == "" || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "Usage: framework-catalog -manifest EDITION.json -source SOURCE_FILE")
		return 2
	}
	manifest, err := os.Open(*manifestPath)
	if err != nil {
		fmt.Fprintln(stderr, "Open manifest:", err)
		return 1
	}
	defer manifest.Close()
	source, err := os.Open(*sourcePath)
	if err != nil {
		fmt.Fprintln(stderr, "Open source:", err)
		return 1
	}
	defer source.Close()

	publication, releaseSHA, sourceSize, err := frameworkcatalog.VerifyPublicationSource(manifest, source)
	if err != nil {
		if errors.Is(err, frameworkcatalog.ErrInvalidPublication) {
			fmt.Fprintln(stderr, "Invalid publication:", err)
		} else {
			fmt.Fprintln(stderr, "Publication verification failed:", err)
		}
		return 1
	}
	receipt := verificationReceipt{
		PublisherCode: publication.PublisherCode,
		FrameworkCode: publication.FrameworkCode,
		EditionCode:   publication.EditionCode,
		ClauseCount:   len(publication.Clauses),
		SourceBytes:   sourceSize,
		SourceSHA256:  publication.SourceSHA256,
		ReleaseSHA256: releaseSHA,
		Published:     false,
	}
	if err := json.NewEncoder(stdout).Encode(receipt); err != nil {
		fmt.Fprintln(stderr, "Write verification receipt:", err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
