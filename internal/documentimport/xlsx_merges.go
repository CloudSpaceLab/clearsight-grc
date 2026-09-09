package documentimport

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

type verticalMerge struct {
	start, end int
	value      string
}

type worksheetMerges map[int][]verticalMerge

func (m worksheetMerges) at(column, row int) *verticalMerge {
	ranges := m[column]
	index := sort.Search(len(ranges), func(i int) bool { return ranges[i].end >= row })
	if index < len(ranges) && ranges[index].start <= row {
		return &ranges[index]
	}
	return nil
}

// Read bounded range metadata without expanding coordinates into cells. Check
// all rectangles for overlap, but inherit only explicit single-column merges.
func worksheetVerticalMerges(ctx context.Context, file *zip.File, policy ExtractionPolicy) (worksheetMerges, error) {
	stream, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	decoder := xml.NewDecoder(stream)
	type rectangle struct{ firstColumn, lastColumn, firstRow, lastRow int }
	ranges := []rectangle{}
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
		if len(ranges) >= min(policy.MaxSections, policy.MaxCells) {
			return nil, limitError("merged range count exceeds extraction bounds")
		}
		ref := ""
		for _, attribute := range start.Attr {
			if attribute.Name.Local == "ref" {
				ref = attribute.Value
			}
		}
		ends := strings.Split(ref, ":")
		if len(ends) != 2 {
			return nil, fmt.Errorf("XLSX contains an invalid merged cell range")
		}
		firstColumn, firstRow, err := mergedCellCoordinates(ends[0], policy)
		if err != nil {
			return nil, err
		}
		lastColumn, lastRow, err := mergedCellCoordinates(ends[1], policy)
		if err != nil {
			return nil, err
		}
		if firstColumn > lastColumn || firstRow > lastRow {
			return nil, fmt.Errorf("XLSX contains a reversed merged cell range")
		}
		ranges = append(ranges, rectangle{firstColumn, lastColumn, firstRow, lastRow})
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].firstRow < ranges[j].firstRow })
	active := []rectangle{}
	result := worksheetMerges{}
	for _, current := range ranges {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		retained := active[:0]
		for _, previous := range active {
			if previous.lastRow < current.firstRow {
				continue
			}
			if previous.firstColumn <= current.lastColumn && current.firstColumn <= previous.lastColumn {
				return nil, fmt.Errorf("XLSX contains overlapping merged cell ranges")
			}
			retained = append(retained, previous)
		}
		// Non-overlapping active rectangles occupy distinct columns, so this
		// sweep retains at most MaxColumns rectangles, regardless of row span.
		active = append(retained, current)
		if current.firstColumn == current.lastColumn && current.firstRow < current.lastRow {
			result[current.firstColumn] = append(result[current.firstColumn], verticalMerge{start: current.firstRow, end: current.lastRow})
		}
	}
	return result, nil
}

func mergedCellCoordinates(reference string, policy ExtractionPolicy) (int, int, error) {
	index, column := 0, 0
	for index < len(reference) && reference[index] >= 'A' && reference[index] <= 'Z' {
		// Check before multiplication to reject arbitrarily long columns.
		if column > (policy.MaxColumns-int(reference[index]-'A'+1))/26 {
			return 0, 0, limitError("merged cell column exceeds extraction bounds")
		}
		column = column*26 + int(reference[index]-'A'+1)
		index++
	}
	if index == 0 || index == len(reference) || reference[index] < '1' || reference[index] > '9' {
		return 0, 0, fmt.Errorf("XLSX contains an invalid merged cell reference")
	}
	for _, digit := range reference[index:] {
		if digit < '0' || digit > '9' {
			return 0, 0, fmt.Errorf("XLSX contains an invalid merged cell reference")
		}
	}
	row, err := strconv.Atoi(reference[index:])
	if err != nil || row > policy.MaxRows || column > policy.MaxColumns {
		return 0, 0, limitError("merged cell range exceeds extraction bounds")
	}
	return column - 1, row, nil
}
