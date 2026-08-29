package documentimport

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"strings"
)

type docxRelationship struct {
	Target  string
	Allowed bool
}

type docxNumberingLevel struct {
	Format string
	Text   string
}

type docxNumbering struct {
	numAbstract map[string]string
	levels      map[string]map[int]docxNumberingLevel
}

func archivePart(files []*zip.File, name string) *zip.File {
	for _, file := range files {
		if file.Name == name {
			return file
		}
	}
	return nil
}

func readDOCXRelationships(ctx context.Context, files []*zip.File) (map[string]docxRelationship, error) {
	file := archivePart(files, "word/_rels/document.xml.rels")
	if file == nil {
		return map[string]docxRelationship{}, nil
	}
	stream, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	values := make(map[string]docxRelationship)
	decoder := xml.NewDecoder(stream)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("DOCX relationships could not be parsed: %w", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "Relationship" {
			continue
		}
		id := strings.TrimSpace(xmlAttribute(start, "Id"))
		target := strings.TrimSpace(xmlAttribute(start, "Target"))
		relType := strings.TrimSpace(xmlAttribute(start, "Type"))
		if id == "" || target == "" || !strings.HasSuffix(relType, "/hyperlink") {
			continue
		}
		values[id] = docxRelationship{Target: target, Allowed: allowedHyperlinkTarget(target)}
	}
	return values, nil
}

func allowedHyperlinkTarget(target string) bool {
	parsed, err := url.Parse(strings.TrimSpace(target))
	if err != nil || parsed.Scheme == "" {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "mailto":
		return true
	default:
		return false
	}
}

func readDOCXNumbering(ctx context.Context, files []*zip.File) (*docxNumbering, error) {
	file := archivePart(files, "word/numbering.xml")
	if file == nil {
		return nil, nil
	}
	stream, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	value := &docxNumbering{
		numAbstract: make(map[string]string),
		levels:      make(map[string]map[int]docxNumberingLevel),
	}
	decoder := xml.NewDecoder(stream)
	currentAbstract := ""
	currentNum := ""
	currentLevel := -1
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("DOCX numbering could not be parsed: %w", err)
		}
		switch item := token.(type) {
		case xml.StartElement:
			switch item.Name.Local {
			case "abstractNum":
				currentAbstract = strings.TrimSpace(xmlAttribute(item, "abstractNumId"))
				currentLevel = -1
				if currentAbstract != "" && value.levels[currentAbstract] == nil {
					value.levels[currentAbstract] = make(map[int]docxNumberingLevel)
				}
			case "lvl":
				currentLevel = parseNonNegativeInt(xmlAttribute(item, "ilvl"), 0)
			case "numFmt":
				if currentAbstract != "" && currentLevel >= 0 {
					level := value.levels[currentAbstract][currentLevel]
					level.Format = strings.TrimSpace(xmlAttribute(item, "val"))
					value.levels[currentAbstract][currentLevel] = level
				}
			case "lvlText":
				if currentAbstract != "" && currentLevel >= 0 {
					level := value.levels[currentAbstract][currentLevel]
					level.Text = strings.TrimSpace(xmlAttribute(item, "val"))
					value.levels[currentAbstract][currentLevel] = level
				}
			case "num":
				currentNum = strings.TrimSpace(xmlAttribute(item, "numId"))
			case "abstractNumId":
				if currentNum != "" {
					value.numAbstract[currentNum] = strings.TrimSpace(xmlAttribute(item, "val"))
				}
			}
		case xml.EndElement:
			switch item.Name.Local {
			case "abstractNum":
				currentAbstract = ""
				currentLevel = -1
			case "num":
				currentNum = ""
			}
		}
	}
	return value, nil
}

func (n *docxNumbering) label(numID string, level int, counters map[string][]int) string {
	if n == nil || strings.TrimSpace(numID) == "" {
		return ""
	}
	abstractID := n.numAbstract[numID]
	definition, ok := n.levels[abstractID][level]
	if !ok {
		return ""
	}
	values := counters[numID]
	if len(values) <= level {
		values = append(values, make([]int, level-len(values)+1)...)
	}
	values[level]++
	for index := level + 1; index < len(values); index++ {
		values[index] = 0
	}
	counters[numID] = values
	if strings.EqualFold(definition.Format, "bullet") {
		if definition.Text != "" {
			return definition.Text
		}
		return "•"
	}
	text := definition.Text
	if text == "" {
		text = "%1."
	}
	for index := 0; index <= level && index < len(values); index++ {
		placeholder := fmt.Sprintf("%%%d", index+1)
		text = strings.ReplaceAll(text, placeholder, fmt.Sprintf("%d", values[index]))
	}
	return strings.TrimSpace(text)
}

func xmlAttribute(start xml.StartElement, local string) string {
	for _, attribute := range start.Attr {
		if attribute.Name.Local == local {
			return attribute.Value
		}
	}
	return ""
}
