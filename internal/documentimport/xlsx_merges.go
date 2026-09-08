package documentimport

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type verticalMerge struct {
	source     string
	start, end int
}

var cellReferencePattern = regexp.MustCompile(`^([A-Z]+)([1-9][0-9]*)$`)

// Read only bounded merge metadata, never expand a merged range into cells.
// Horizontal merges do not establish row ownership and are not propagated.
func worksheetVerticalMerges(ctx context.Context, file *zip.File, policy ExtractionPolicy) (map[int][]verticalMerge, error) {
	stream, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	decoder := xml.NewDecoder(stream)
	result := map[int][]verticalMerge{}
	count := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "mergeCell" {
			continue
		}
		count++
		if count > policy.MaxSections {
			return nil, limitError("merged range count exceeds %d", policy.MaxSections)
		}
		ref := ""
		for _, a := range start.Attr {
			if a.Name.Local == "ref" {
				ref = a.Value
			}
		}
		ends := strings.Split(ref, ":")
		if len(ends) != 2 {
			return nil, fmt.Errorf("invalid merged cell range")
		}
		a, b := cellReferencePattern.FindStringSubmatch(ends[0]), cellReferencePattern.FindStringSubmatch(ends[1])
		if a == nil || b == nil {
			return nil, fmt.Errorf("invalid merged cell reference")
		}
		first, e1 := strconv.Atoi(a[2])
		last, e2 := strconv.Atoi(b[2])
		column := cellColumn(ends[0])
		if e1 != nil || e2 != nil || first > last || last > policy.MaxRows || cellColumn(ends[1]) >= policy.MaxColumns {
			return nil, limitError("merged cell range exceeds extraction bounds")
		}
		if a[1] != b[1] || first == last {
			continue
		}
		for _, existing := range result[column] {
			if first <= existing.end && last >= existing.start {
				return nil, fmt.Errorf("overlapping merged cell ranges")
			}
		}
		result[column] = append(result[column], verticalMerge{source: ends[0], start: first, end: last})
	}
	return result, nil
}
