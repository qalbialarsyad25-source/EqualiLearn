package document

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestExtractTextFromDocument_Text(t *testing.T) {
	data := []byte("Photosynthesis is the biological process used by plants to convert light energy into chemical energy.")
	content, err := ExtractTextFromDocument("photosynthesis.txt", data, "text/plain")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if content.FileType != "txt" {
		t.Errorf("expected fileType 'txt', got %q", content.FileType)
	}

	if content.WordCount < 5 {
		t.Errorf("expected wordCount >= 5, got %d", content.WordCount)
	}

	if content.RawText != string(data) {
		t.Errorf("expected rawText %q, got %q", string(data), content.RawText)
	}
}

func TestExtractTextFromDocument_PDF(t *testing.T) {
	pdfData := []byte("%PDF-1.7 sample content")
	content, err := ExtractTextFromDocument("lecture.pdf", pdfData, "application/pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if content.FileType != "pdf" {
		t.Errorf("expected fileType 'pdf', got %q", content.FileType)
	}

	if content.ContentType != "application/pdf" {
		t.Errorf("expected contentType 'application/pdf', got %q", content.ContentType)
	}
}

func TestExtractFromPPTX_ValidZip(t *testing.T) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	slide1Content := `<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
		<p:cSld>
			<p:spTree>
				<p:sp>
					<p:txBody>
						<a:p><a:r><a:t>Introduction to Machine Learning</a:t></a:r></a:p>
						<a:p><a:r><a:t>Supervised vs Unsupervised Learning</a:t></a:r></a:p>
					</p:txBody>
				</p:sp>
			</p:spTree>
		</p:cSld>
	</p:sld>`

	w1, err := zipWriter.Create("ppt/slides/slide1.xml")
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}
	_, _ = w1.Write([]byte(slide1Content))

	slide2Content := `<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
		<p:cSld>
			<p:spTree>
				<p:sp>
					<p:txBody>
						<a:p><a:r><a:t>Neural Networks and Deep Learning</a:t></a:r></a:p>
					</p:txBody>
				</p:sp>
			</p:spTree>
		</p:cSld>
	</p:sld>`

	w2, err := zipWriter.Create("ppt/slides/slide2.xml")
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}
	_, _ = w2.Write([]byte(slide2Content))

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	content, err := ExtractFromPPTX(buf.Bytes())
	if err != nil {
		t.Fatalf("unexpected error extracting PPTX: %v", err)
	}

	if content.SlideCount != 2 {
		t.Errorf("expected 2 slides, got %d", content.SlideCount)
	}

	if !bytes.Contains([]byte(content.RawText), []byte("Introduction to Machine Learning")) {
		t.Errorf("expected slide text to contain 'Introduction to Machine Learning'")
	}

	if !bytes.Contains([]byte(content.RawText), []byte("Neural Networks and Deep Learning")) {
		t.Errorf("expected slide text to contain 'Neural Networks and Deep Learning'")
	}
}
