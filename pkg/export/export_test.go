package export

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestGenerateTextAndMarkdown(t *testing.T) {
	payload := ExportPayload{
		Title:          "Introduction to Machine Learning",
		Summary:        "Machine learning is a subset of artificial intelligence that enables computers to learn from data.",
		KeyPoints:      []string{"Supervised learning uses labeled data", "Unsupervised learning finds hidden patterns", "Neural networks mimic brain structures"},
		Explanation:    "Machine learning algorithms build a mathematical model based on sample data to make predictions.",
		TranscriptText: "Hello everyone, today we will talk about machine learning fundamentals.",
		Language:       "en",
		DetailLevel:    "balanced",
		TargetAudience: "student",
		SourceType:     "Speech-to-Text",
		CreatedAt:      time.Now(),
	}

	txt := GenerateText(payload)
	if len(txt) == 0 {
		t.Fatal("expected non-empty text export")
	}
	txtStr := string(txt)
	if !strings.Contains(txtStr, "Introduction to Machine Learning") {
		t.Errorf("expected title in text output")
	}
	if !strings.Contains(txtStr, "Supervised learning uses labeled data") {
		t.Errorf("expected key point in text output")
	}

	md := GenerateMarkdown(payload)
	if len(md) == 0 {
		t.Fatal("expected non-empty markdown export")
	}
	mdStr := string(md)
	if !strings.Contains(mdStr, "# Introduction to Machine Learning") {
		t.Errorf("expected markdown header")
	}
}

func TestGeneratePDF(t *testing.T) {
	payload := ExportPayload{
		Title:          "Revolusi Industri 4.0 dan Kecerdasan Buatan",
		Summary:        "Materi ini membahas implementasi teknologi AI dalam otomatisasi proses produksi dan pendidikan modern.",
		KeyPoints:      []string{"Otomasi cerdas meningkatkan efisiensi operasional", "Pentingnya literasi digital di era modern", "Kolaborasi manusia dan AI menciptakan nilai tambah"},
		Explanation:    "Revolusi industri 4.0 menggabungkan teknologi fisik dan digital seperti IoT, Big Data, dan Cloud Computing.",
		TranscriptText: "Selamat pagi rekan-rekan mahasiswa, pada sesi kali ini kita membahas revolusi industri 4.0.",
		Language:       "id",
		DetailLevel:    "detailed",
		TargetAudience: "student",
		SourceType:     "Speech-to-Text",
		CreatedAt:      time.Now(),
	}

	pdf := GeneratePDF(payload)
	if len(pdf) == 0 {
		t.Fatal("expected non-empty PDF bytes")
	}

	// Verify standard PDF header and trailer
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) {
		t.Errorf("expected PDF header %%PDF-1.4")
	}
	if !bytes.Contains(pdf, []byte("%%EOF")) {
		t.Errorf("expected PDF EOF marker")
	}
	if !bytes.Contains(pdf, []byte("xref")) {
		t.Errorf("expected xref table in PDF")
	}
}
