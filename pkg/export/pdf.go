package export

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

// PDF Document Constants (A4 size in standard 72 DPI points)
const (
	pageWidth   = 595.28
	pageHeight  = 841.89
	marginLeft  = 50.0
	marginRight = 545.0
	usableWidth = marginRight - marginLeft // 495.0
	marginTop   = 780.0
	marginBottom = 65.0
)

// pdfPage represents an individual page's stream operations
type pdfPage struct {
	stream bytes.Buffer
}

// GeneratePDF creates a styled, professional PDF document from ExportPayload in pure Go.
func GeneratePDF(p ExportPayload) []byte {
	created := p.CreatedAt
	if created.IsZero() {
		created = time.Now()
	}

	var pages []*pdfPage

	newPage := func() *pdfPage {
		pg := &pdfPage{}
		pages = append(pages, pg)
		return pg
	}

	currentPage := newPage()
	curY := marginTop

	// Helper to escape PDF string characters
	escapePDFText := func(s string) string {
		s = strings.ReplaceAll(s, "\\", "\\\\")
		s = strings.ReplaceAll(s, "(", "\\(")
		s = strings.ReplaceAll(s, ")", "\\)")
		// Replace non-ASCII / Unicode quotes and bullets with standard equivalents
		s = strings.ReplaceAll(s, "“", "\"")
		s = strings.ReplaceAll(s, "”", "\"")
		s = strings.ReplaceAll(s, "‘", "'")
		s = strings.ReplaceAll(s, "’", "'")
		s = strings.ReplaceAll(s, "—", "-")
		s = strings.ReplaceAll(s, "–", "-")
		s = strings.ReplaceAll(s, "•", "*")
		return s
	}

	// Approximate width estimation for Helvetica
	calcLineWidth := func(text string, fontSize float64) float64 {
		return float64(len(text)) * fontSize * 0.52
	}

	// Helper to split a long paragraph into lines that fit within max width
	wrapText := func(text string, fontSize float64, maxWidth float64) []string {
		words := strings.Fields(text)
		if len(words) == 0 {
			return nil
		}

		var lines []string
		var curLine strings.Builder

		for _, word := range words {
			testStr := word
			if curLine.Len() > 0 {
				testStr = curLine.String() + " " + word
			}

			if calcLineWidth(testStr, fontSize) > maxWidth && curLine.Len() > 0 {
				lines = append(lines, curLine.String())
				curLine.Reset()
				curLine.WriteString(word)
			} else {
				if curLine.Len() > 0 {
					curLine.WriteString(" ")
				}
				curLine.WriteString(word)
			}
		}

		if curLine.Len() > 0 {
			lines = append(lines, curLine.String())
		}

		return lines
	}

	ensureSpace := func(needed float64) {
		if curY-needed < marginBottom {
			currentPage = newPage()
			curY = marginTop
		}
	}

	// 1. Top Header Brand Bar (Page 1)
	currentPage.stream.WriteString(fmt.Sprintf("0.12 0.45 0.90 rg\n%.2f %.2f %.2f %.2f re f\n", marginLeft, curY-4, usableWidth, 2.0))
	curY -= 16

	// Brand text
	currentPage.stream.WriteString(fmt.Sprintf("BT /F2 8 Tf 0.4 0.4 0.4 rg %.2f %.2f Td (%s) Tj ET\n",
		marginLeft, curY, escapePDFText("EQUILILEARN - AI EDUCATIONAL LEARNING ASSISTANT")))
	curY -= 24

	// 2. Document Title
	titleLines := wrapText(p.Title, 16.0, usableWidth)
	for _, tl := range titleLines {
		ensureSpace(22)
		currentPage.stream.WriteString(fmt.Sprintf("BT /F2 16 Tf 0.1 0.1 0.15 rg %.2f %.2f Td (%s) Tj ET\n",
			marginLeft, curY, escapePDFText(tl)))
		curY -= 20
	}
	curY -= 6

	// 3. Metadata Subtitle Row
	metaText := fmt.Sprintf("Source: %s  |  Date: %s  |  Language: %s  |  Detail: %s",
		p.SourceType,
		created.Format("02 Jan 2006, 15:04"),
		strings.ToUpper(p.Language),
		strings.Title(p.DetailLevel),
	)
	currentPage.stream.WriteString(fmt.Sprintf("BT /F3 9 Tf 0.45 0.45 0.50 rg %.2f %.2f Td (%s) Tj ET\n",
		marginLeft, curY, escapePDFText(metaText)))
	curY -= 16

	// Divider
	currentPage.stream.WriteString(fmt.Sprintf("0.85 0.88 0.92 rg\n%.2f %.2f %.2f %.2f re f\n", marginLeft, curY, usableWidth, 1.0))
	curY -= 20

	// Helper to draw section header
	drawSectionHeader := func(title string) {
		ensureSpace(35)
		// Section Pill Background
		currentPage.stream.WriteString(fmt.Sprintf("0.93 0.95 0.98 rg\n%.2f %.2f %.2f %.2f re f\n", marginLeft, curY-3, usableWidth, 18.0))
		currentPage.stream.WriteString(fmt.Sprintf("0.15 0.40 0.85 rg\n%.2f %.2f %.2f %.2f re f\n", marginLeft, curY-3, 3.5, 18.0))
		currentPage.stream.WriteString(fmt.Sprintf("BT /F2 11 Tf 0.12 0.25 0.60 rg %.2f %.2f Td (%s) Tj ET\n",
			marginLeft+10, curY+1, escapePDFText(strings.ToUpper(title))))
		curY -= 24
	}

	// Helper to render multiline paragraph
	renderParagraph := func(text string, fontSize float64, lineSpacing float64) {
		paras := strings.Split(text, "\n")
		for _, para := range paras {
			trimmed := strings.TrimSpace(para)
			if trimmed == "" {
				curY -= lineSpacing * 0.6
				continue
			}

			lines := wrapText(trimmed, fontSize, usableWidth)
			for _, line := range lines {
				ensureSpace(lineSpacing)
				currentPage.stream.WriteString(fmt.Sprintf("BT /F1 %.1f Tf 0.20 0.22 0.25 rg %.2f %.2f Td (%s) Tj ET\n",
					fontSize, marginLeft, curY, escapePDFText(line)))
				curY -= lineSpacing
			}
			curY -= 4
		}
	}

	// 4. Executive Summary Section
	if p.Summary != "" {
		drawSectionHeader("Executive Summary")
		renderParagraph(p.Summary, 10.0, 14.0)
		curY -= 12
	}

	// 5. Key Takeaways Section
	if len(p.KeyPoints) > 0 {
		drawSectionHeader("Key Takeaways")
		for i, kp := range p.KeyPoints {
			trimmed := strings.TrimSpace(kp)
			if trimmed == "" {
				continue
			}

			kpLines := wrapText(trimmed, 9.5, usableWidth-24)
			for j, line := range kpLines {
				ensureSpace(14)
				if j == 0 {
					// Bullet icon / Number
					currentPage.stream.WriteString(fmt.Sprintf("BT /F2 9.5 Tf 0.15 0.45 0.85 rg %.2f %.2f Td (%d.) Tj ET\n",
						marginLeft+4, curY, i+1))
					currentPage.stream.WriteString(fmt.Sprintf("BT /F1 9.5 Tf 0.18 0.20 0.24 rg %.2f %.2f Td (%s) Tj ET\n",
						marginLeft+22, curY, escapePDFText(line)))
				} else {
					currentPage.stream.WriteString(fmt.Sprintf("BT /F1 9.5 Tf 0.18 0.20 0.24 rg %.2f %.2f Td (%s) Tj ET\n",
						marginLeft+22, curY, escapePDFText(line)))
				}
				curY -= 13.5
			}
			curY -= 4
		}
		curY -= 10
	}

	// 6. In-Depth Explanation Section
	if p.Explanation != "" {
		drawSectionHeader("Conceptual Explanation & Key Insights")
		renderParagraph(p.Explanation, 9.5, 13.5)
		curY -= 12
	}

	// 7. Original Transcript Section (if present)
	if p.TranscriptText != "" {
		drawSectionHeader("Speech Transcription Content")
		renderParagraph(p.TranscriptText, 9.0, 12.5)
		curY -= 12
	}

	// 8. Add Page Footers across all generated pages
	totalPages := len(pages)
	for i, pg := range pages {
		pg.stream.WriteString(fmt.Sprintf("0.85 0.88 0.92 rg\n%.2f %.2f %.2f %.2f re f\n", marginLeft, marginBottom+12, usableWidth, 0.75))
		footerLeft := "EquiliLearn - AI Powered Inclusive Learning"
		footerRight := fmt.Sprintf("Page %d of %d", i+1, totalPages)
		pg.stream.WriteString(fmt.Sprintf("BT /F3 8 Tf 0.50 0.50 0.55 rg %.2f %.2f Td (%s) Tj ET\n",
			marginLeft, marginBottom, escapePDFText(footerLeft)))
		pg.stream.WriteString(fmt.Sprintf("BT /F3 8 Tf 0.50 0.50 0.55 rg %.2f %.2f Td (%s) Tj ET\n",
			marginRight-50, marginBottom, escapePDFText(footerRight)))
	}

	// Assemble final PDF document
	return compilePDFDocument(pages)
}

// compilePDFDocument builds the raw PDF 1.4 byte structure including object catalog, pages, fonts, xref table, and trailer.
func compilePDFDocument(pages []*pdfPage) []byte {
	var buf bytes.Buffer
	var offsets []int

	// 1. PDF Header
	buf.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")

	// Helper to track byte offsets for XRef
	addObject := func(content string) int {
		offsets = append(offsets, buf.Len())
		objNum := len(offsets)
		buf.WriteString(fmt.Sprintf("%d 0 obj\n%s\nendobj\n", objNum, content))
		return objNum
	}

	// Obj 1: Catalog (will point to Obj 2: Pages tree)
	// Obj 2: Pages tree
	// Obj 3: Font Helvetica (F1)
	// Obj 4: Font Helvetica-Bold (F2)
	// Obj 5: Font Helvetica-Oblique (F3)

	fontF1Obj := 3
	fontF2Obj := 4
	fontF3Obj := 5

	numPages := len(pages)
	pageObjStart := 6
	streamObjStart := pageObjStart + numPages

	// Pre-create kids list
	var kidsList strings.Builder
	for i := 0; i < numPages; i++ {
		kidsList.WriteString(fmt.Sprintf("%d 0 R ", pageObjStart+i))
	}

	// Obj 1: Catalog
	addObject("<< /Type /Catalog /Pages 2 0 R >>")

	// Obj 2: Pages
	addObject(fmt.Sprintf("<< /Type /Pages /Kids [ %s] /Count %d >>", kidsList.String(), numPages))

	// Obj 3: Font F1 (Helvetica)
	addObject("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")

	// Obj 4: Font F2 (Helvetica-Bold)
	addObject("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")

	// Obj 5: Font F3 (Helvetica-Oblique)
	addObject("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Oblique /Encoding /WinAnsiEncoding >>")

	// Create Page Objects
	for i := 0; i < numPages; i++ {
		streamObjNum := streamObjStart + i
		pageContent := fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Contents %d 0 R /Resources << /Font << /F1 %d 0 R /F2 %d 0 R /F3 %d 0 R >> >> >>",
			pageWidth, pageHeight, streamObjNum, fontF1Obj, fontF2Obj, fontF3Obj)
		addObject(pageContent)
	}

	// Create Page Stream Content Objects
	for _, page := range pages {
		streamBytes := page.stream.Bytes()
		streamHeader := fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(streamBytes), string(streamBytes))
		addObject(streamHeader)
	}

	// Cross-reference table
	xrefOffset := buf.Len()
	buf.WriteString(fmt.Sprintf("xref\n0 %d\n", len(offsets)+1))
	buf.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
	}

	// Trailer
	buf.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets)+1, xrefOffset))

	return buf.Bytes()
}
