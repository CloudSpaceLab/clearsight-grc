package documentimport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var ErrLegacyOfficeConversion = errors.New("legacy office conversion failed")

type LegacyOfficeConverter struct {
	Executable     string
	Timeout        time.Duration
	MaxInputBytes  int64
	MaxOutputBytes int64
}

func (c LegacyOfficeConverter) Enabled() bool {
	return strings.TrimSpace(c.Executable) != ""
}

func (c LegacyOfficeConverter) ConvertXLS(ctx context.Context, fileName string, data []byte) ([]byte, error) {
	if !c.Enabled() {
		return nil, errors.Join(ErrLegacyOfficeConversion, errors.New("legacy office converter is disabled"))
	}
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	if c.MaxInputBytes <= 0 {
		c.MaxInputBytes = 20 << 20
	}
	if c.MaxOutputBytes <= 0 {
		c.MaxOutputBytes = 64 << 20
	}
	if int64(len(data)) == 0 || int64(len(data)) > c.MaxInputBytes {
		return nil, errors.Join(ErrLegacyOfficeConversion, errors.New("legacy office input is empty or exceeds the configured limit"))
	}
	if strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))) != ".xls" {
		return nil, errors.Join(ErrLegacyOfficeConversion, errors.New("only legacy .xls conversion is supported"))
	}

	root, err := os.MkdirTemp("", "clearsight-office-")
	if err != nil {
		return nil, errors.Join(ErrLegacyOfficeConversion, err)
	}
	defer os.RemoveAll(root)
	inputDir := filepath.Join(root, "input")
	outputDir := filepath.Join(root, "output")
	profileDir := filepath.Join(root, "profile")
	for _, directory := range []string{inputDir, outputDir, profileDir} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			return nil, errors.Join(ErrLegacyOfficeConversion, err)
		}
	}
	inputPath := filepath.Join(inputDir, "source.xls")
	if err := os.WriteFile(inputPath, data, 0o600); err != nil {
		return nil, errors.Join(ErrLegacyOfficeConversion, err)
	}

	boundedCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	profileURI := "file://" + filepath.ToSlash(profileDir)
	command := exec.CommandContext(boundedCtx, c.Executable,
		"--headless", "--norestore", "--nodefault", "--nolockcheck", "--nofirststartwizard",
		"-env:UserInstallation="+profileURI,
		"--convert-to", "xlsx", "--outdir", outputDir, inputPath,
	)
	command.Dir = root
	command.Env = minimalOfficeEnvironment()
	output, err := command.CombinedOutput()
	if boundedCtx.Err() != nil {
		return nil, errors.Join(ErrLegacyOfficeConversion, boundedCtx.Err())
	}
	if err != nil {
		return nil, errors.Join(ErrLegacyOfficeConversion, fmt.Errorf("converter exited unsuccessfully: %w", err))
	}
	if len(output) > 64<<10 {
		return nil, errors.Join(ErrLegacyOfficeConversion, errors.New("converter diagnostic output exceeded the configured limit"))
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, errors.Join(ErrLegacyOfficeConversion, err)
	}
	if len(entries) != 1 || entries[0].IsDir() || strings.ToLower(filepath.Ext(entries[0].Name())) != ".xlsx" {
		return nil, errors.Join(ErrLegacyOfficeConversion, errors.New("converter did not produce exactly one xlsx output"))
	}
	outputPath := filepath.Join(outputDir, entries[0].Name())
	info, err := os.Lstat(outputPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > c.MaxOutputBytes {
		return nil, errors.Join(ErrLegacyOfficeConversion, errors.New("converted xlsx is invalid or exceeds the configured output limit"))
	}
	file, err := os.Open(outputPath)
	if err != nil {
		return nil, errors.Join(ErrLegacyOfficeConversion, err)
	}
	defer file.Close()
	converted, err := io.ReadAll(io.LimitReader(file, c.MaxOutputBytes+1))
	if err != nil {
		return nil, errors.Join(ErrLegacyOfficeConversion, err)
	}
	if int64(len(converted)) > c.MaxOutputBytes {
		return nil, errors.Join(ErrLegacyOfficeConversion, errors.New("converted xlsx exceeded the configured output limit"))
	}
	return converted, nil
}

func minimalOfficeEnvironment() []string {
	result := make([]string, 0, 4)
	for _, name := range []string{"PATH", "HOME", "LANG", "LC_ALL"} {
		if value, ok := os.LookupEnv(name); ok && strings.TrimSpace(value) != "" {
			result = append(result, name+"="+value)
		}
	}
	return result
}
