package documentimport

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestLegacyOfficeConverterDisabledAndInputLimit(t *testing.T) {
	if _, err := (LegacyOfficeConverter{}).ConvertXLS(t.Context(), "legacy.xls", []byte("xls")); !errors.Is(err, ErrLegacyOfficeConversion) {
		t.Fatalf("disabled converter error = %v", err)
	}
	converter := LegacyOfficeConverter{Executable: "/does/not/matter", MaxInputBytes: 2, MaxOutputBytes: 10, Timeout: time.Second}
	if _, err := converter.ConvertXLS(t.Context(), "legacy.xls", []byte("too large")); !errors.Is(err, ErrLegacyOfficeConversion) {
		t.Fatalf("input limit error = %v", err)
	}
}

func TestLegacyOfficeConverterTimeout(t *testing.T) {
	executable := writeConverterExecutable(t)
	converter := LegacyOfficeConverter{Executable: executable, MaxInputBytes: 1024, MaxOutputBytes: 1024, Timeout: 10 * time.Millisecond}
	_, err := converter.ConvertXLS(t.Context(), "legacy.xls", []byte("timeout"))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout error = %v", err)
	}
}

func TestLegacyOfficeConverterOutputLimitAndSingleArtifact(t *testing.T) {
	cases := []struct {
		name string
		mode string
	}{
		{name: "too-large", mode: "too-large"},
		{name: "multiple", mode: "multiple"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			executable := writeConverterExecutable(t)
			converter := LegacyOfficeConverter{Executable: executable, MaxInputBytes: 1024, MaxOutputBytes: 8, Timeout: time.Second}
			_, err := converter.ConvertXLS(t.Context(), "legacy.xls", []byte(tc.mode))
			if !errors.Is(err, ErrLegacyOfficeConversion) {
				t.Fatalf("expected bounded conversion failure, got %v", err)
			}
		})
	}
}

func TestLegacyOfficeConverterReturnsOnlyBoundedXLSX(t *testing.T) {
	executable := writeConverterExecutable(t)
	converter := LegacyOfficeConverter{Executable: executable, MaxInputBytes: 1024, MaxOutputBytes: 1024, Timeout: time.Second}
	converted, err := converter.ConvertXLS(t.Context(), "legacy.xls", []byte("valid"))
	if err != nil {
		t.Fatal(err)
	}
	if string(converted) != "xlsx-bytes" {
		t.Fatalf("converted bytes = %q", converted)
	}
}

func writeConverterExecutable(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "main.go")
	executablePath := filepath.Join(directory, "fake-libreoffice")
	if runtime.GOOS == "windows" {
		executablePath += ".exe"
	}
	source := `package main

import (
	"os"
	"path/filepath"
	"time"
)

func main() {
	outputDir := ""
	inputPath := ""
	for index, value := range os.Args[1:] {
		if value == "--outdir" && index+2 < len(os.Args) {
			outputDir = os.Args[index+2]
		}
		inputPath = value
	}
	input, err := os.ReadFile(inputPath)
	if err != nil || outputDir == "" {
		os.Exit(2)
	}
	switch string(input) {
	case "timeout":
		time.Sleep(time.Second)
	case "too-large":
		_ = os.WriteFile(filepath.Join(outputDir, "source.xlsx"), []byte("0123456789abcdef"), 0600)
	case "multiple":
		_ = os.WriteFile(filepath.Join(outputDir, "source.xlsx"), []byte("one"), 0600)
		_ = os.WriteFile(filepath.Join(outputDir, "second.xlsx"), []byte("two"), 0600)
	case "valid":
		_ = os.WriteFile(filepath.Join(outputDir, "source.xlsx"), []byte("xlsx-bytes"), 0600)
	default:
		os.Exit(3)
	}
}
`
	if err := os.WriteFile(sourcePath, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "build", "-o", executablePath, sourcePath)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build fake office converter: %v\n%s", err, output)
	}
	return executablePath
}
