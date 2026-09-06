package document

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ExtractedContent represents the result of document parsing.
type ExtractedContent struct {
	RawText     string
	SlideCount  int
	WordCount   int
	FileType    string
	ContentType string
}

// ExtractTextFromDocument extracts textual content from supported file formats (pptx, txt, md, etc.).
func ExtractTextFromDocument(filename string, data []byte, contentType string) (*ExtractedContent, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	switch {
	case ext == ".pptx" || strings.Contains(contentType, "presentationml.presentation"):
		return ExtractFromPPTX(data)
	case ext == ".txt" || ext == ".md" || ext == ".csv" || ext == ".json" || strings.HasPrefix(contentType, "text/"):
		text := strings.TrimSpace(string(data))
		wordCount := len(strings.Fields(text))
		return &ExtractedContent{
			RawText:     text,
			SlideCount:  0,
			WordCount:   wordCount,
			FileType:    strings.TrimPrefix(ext, "."),
			ContentType: "text/plain",
		}, nil
	case ext == ".pdf" || contentType == "application/pdf":
		// For PDF, raw binary is sent directly to Gemini multimodal API
		return &ExtractedContent{
			RawText:     "",
			SlideCount:  0,
			WordCount:   0,
			FileType:    "pdf",
			ContentType: "application/pdf",
		}, nil
	default:
		// Attempt plain text extraction if readable UTF-8
		text := strings.TrimSpace(string(data))
		if isPrintableText(text) {
			return &ExtractedContent{
				RawText:     text,
				SlideCount:  0,
				WordCount:   len(strings.Fields(text)),
				FileType:    strings.TrimPrefix(ext, "."),
				ContentType: "text/plain",
			}, nil
		}
		return nil, fmt.Errorf("unsupported file format %q. Supported formats: .pdf, .pptx, .txt, .md", ext)
	}
}

// ExtractFromPPTX extracts slide-by-slide text, titles, and speaker notes from a PPTX file.
func ExtractFromPPTX(data []byte) (*ExtractedContent, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open pptx archive: %w", err)
	}

	// Find and sort all slide files: ppt/slides/slide1.xml, slide2.xml, ...
	slideRegex := regexp.MustCompile(`^ppt/slides/slide(\d+)\.xml$`)
	type slideEntry struct {
		num  int
		file *zip.File
	}
	var slides []slideEntry

	for _, file := range reader.File {
		matches := slideRegex.FindStringSubmatch(file.Name)
		if len(matches) == 2 {
			num, _ := strconv.Atoi(matches[1])
			slides = append(slides, slideEntry{num: num, file: file})
		}
	}

	sort.Slice(slides, func(i, j int) bool {
		return slides[i].num < slides[j].num
	})

	var resultBuilder strings.Builder
	totalWords := 0

	for _, slide := range slides {
		slideText, err := extractTextFromZipFile(slide.file)
		if err != nil {
			continue
		}

		trimmed := strings.TrimSpace(slideText)
		if trimmed != "" {
			resultBuilder.WriteString(fmt.Sprintf("\n--- [SLIDE %d] ---\n", slide.num))
			resultBuilder.WriteString(trimmed)
			resultBuilder.WriteString("\n")
			totalWords += len(strings.Fields(trimmed))
		}
	}

	finalText := strings.TrimSpace(resultBuilder.String())
	if finalText == "" {
		finalText = "(No text found in presentation slides)"
	}

	return &ExtractedContent{
		RawText:     finalText,
		SlideCount:  len(slides),
		WordCount:   totalWords,
		FileType:    "pptx",
		ContentType: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	}, nil
}

// extractTextFromZipFile extracts plain text from OpenXML slide files by decoding XML tokens.
func extractTextFromZipFile(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var sb strings.Builder
	var inTextTag bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			// <a:t> represents text runs in DrawingML / OpenXML presentations
			if elem.Name.Local == "t" {
				inTextTag = true
			} else if elem.Name.Local == "p" || elem.Name.Local == "br" {
				// New paragraph or line break
				sb.WriteString("\n")
			}
		case xml.EndElement:
			if elem.Name.Local == "t" {
				inTextTag = false
			}
		case xml.CharData:
			if inTextTag {
				text := strings.TrimSpace(string(elem))
				if text != "" {
					sb.WriteString(text + " ")
				}
			}
		}
	}

	// Clean up multi-newlines
	lines := strings.Split(sb.String(), "\n")
	var cleanLines []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			cleanLines = append(cleanLines, trimmed)
		}
	}

	return strings.Join(cleanLines, "\n"), nil
}

func isPrintableText(s string) bool {
	if len(s) == 0 {
		return false
	}
	sampleLen := len(s)
	if sampleLen > 512 {
		sampleLen = 512
	}
	for i := 0; i < sampleLen; i++ {
		b := s[i]
		if b < 32 && b != '\t' && b != '\n' && b != '\r' {
			return false
		}
	}
	return true
}
