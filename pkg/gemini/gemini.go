package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"google.golang.org/genai"
)

// GeminiSummaryRequest defines parameters for generating structured summaries from text or documents.
type GeminiSummaryRequest struct {
	InlineData     []byte // Raw PDF or document bytes
	MIMEType       string // e.g. "application/pdf"
	TextContent    string // Extracted text or slides
	FileName       string // Name of the uploaded file
	Language       string // "id" (Indonesian), "en" (English), etc.
	DetailLevel    string // "brief", "balanced", "detailed"
	TargetAudience string // "student", "beginner", "professional", "general"
}

// GeminiSummaryResult represents the structured response parsed from Gemini.
type GeminiSummaryResult struct {
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	KeyPoints   []string `json:"key_points"`
	Explanation string   `json:"explanation"`
	Model       string   `json:"model"`
	TokenCount  int      `json:"token_count,omitempty"`
}

// IGeminiClient defines the interface for Gemini AI interactions.
type IGeminiClient interface {
	SummarizeContent(ctx context.Context, req GeminiSummaryRequest) (*GeminiSummaryResult, error)
}

// ==========================================
// Official Google GenAI Gemini Client
// ==========================================

type GenAIClient struct {
	client *genai.Client
	model  string
}

func NewGenAIClient(apiKey, modelName string) (*GenAIClient, error) {
	ctx := context.Background()
	var clientConfig *genai.ClientConfig
	if apiKey != "" {
		clientConfig = &genai.ClientConfig{
			APIKey: apiKey,
		}
	}

	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	if modelName == "" {
		modelName = "gemini-3.7-flash"
	}

	return &GenAIClient{
		client: client,
		model:  modelName,
	}, nil
}

func (g *GenAIClient) SummarizeContent(ctx context.Context, req GeminiSummaryRequest) (*GeminiSummaryResult, error) {
	langName := "Indonesian (Bahasa Indonesia)"
	if strings.ToLower(req.Language) == "en" {
		langName = "English"
	}

	detailInstruction := "Provide a clear, balanced summary with essential takeaways."
	switch strings.ToLower(req.DetailLevel) {
	case "brief":
		detailInstruction = "Provide a concise, high-level summary emphasizing only the most critical ideas."
	case "detailed":
		detailInstruction = "Provide an in-depth, comprehensive summary capturing core concepts, supporting arguments, and nuanced insights."
	}

	audienceInstruction := "Make the explanations accessible for students and general learners."
	switch strings.ToLower(req.TargetAudience) {
	case "beginner":
		audienceInstruction = "Explain concepts in simple, easy-to-understand terms with everyday analogies, avoiding overly dense jargon."
	case "student":
		audienceInstruction = "Structure the explanation to optimize student learning, retention, and exam preparation."
	case "professional":
		audienceInstruction = "Use professional terminology and focus on actionable insights and practical applications."
	}

	systemPrompt := fmt.Sprintf(`You are an expert educational AI assistant and learning summarizer for EquiliLearn.
Your objective is to analyze the provided material (document, presentation slides, or text) and produce a high-impact, educational summary and explanation.

INSTRUCTIONS:
1. Target Language: Output ALL fields strictly in %s.
2. Detail Level: %s
3. Audience Persona: %s
4. You MUST respond with ONLY a valid JSON object adhering to this exact schema (no additional commentary, no markdown codeblock wraps if possible, just raw JSON):
{
  "title": "A concise, engaging title for this document/topic",
  "summary": "A cohesive 2 to 4 paragraph overview explaining what the material covers, its central theme, and conclusions.",
  "key_points": [
    "Key takeaway point 1 with concise explanation",
    "Key takeaway point 2 with concise explanation",
    "Key takeaway point 3 with concise explanation",
    "Key takeaway point 4 with concise explanation",
    "Key takeaway point 5 with concise explanation"
  ],
  "explanation": "A structured, detailed, easy-to-digest conceptual explanation of the material. Break down complex ideas into step-by-step intuition, why it matters, and real-world examples."
}`, langName, detailInstruction, audienceInstruction)

	var parts []*genai.Part
	parts = append(parts, genai.NewPartFromText(systemPrompt))

	// Include multimodal binary data if PDF
	if len(req.InlineData) > 0 && req.MIMEType != "" {
		parts = append(parts, genai.NewPartFromBytes(req.InlineData, req.MIMEType))
		parts = append(parts, genai.NewPartFromText(fmt.Sprintf("Analyze the attached document (%s) and generate the requested JSON summary.", req.FileName)))
	} else if req.TextContent != "" {
		parts = append(parts, genai.NewPartFromText(fmt.Sprintf("Material text/slides to summarize:\n\n%s\n\nGenerate the requested JSON summary.", req.TextContent)))
	} else {
		return nil, fmt.Errorf("no document content or text provided for summarization")
	}

	contents := []*genai.Content{
		{
			Role:  "user",
			Parts: parts,
		},
	}

	resp, err := g.client.Models.GenerateContent(ctx, g.model, contents, nil)
	if err != nil {
		return nil, fmt.Errorf("gemini generate content failed: %w", err)
	}

	rawText := resp.Text()
	result, err := parseGeminiJSONResponse(rawText)
	if err != nil {
		// Fallback if model output unstructured text
		result = &GeminiSummaryResult{
			Title:       extractTitleFromFilename(req.FileName),
			Summary:     rawText,
			KeyPoints:   []string{"Summary generated by Gemini AI"},
			Explanation: rawText,
		}
	}

	result.Model = g.model
	if resp.UsageMetadata != nil {
		result.TokenCount = int(resp.UsageMetadata.TotalTokenCount)
	}

	return result, nil
}

// ==========================================
// Mock Gemini Client (Offline/Dev)
// ==========================================

type MockGeminiClient struct {
	model string
}

func NewMockGeminiClient() *MockGeminiClient {
	return &MockGeminiClient{
		model: "gemini-3.7-flash (mock-mode)",
	}
}

func (m *MockGeminiClient) SummarizeContent(ctx context.Context, req GeminiSummaryRequest) (*GeminiSummaryResult, error) {
	// Simulate AI thinking latency
	select {
	case <-time.After(150 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	title := extractTitleFromFilename(req.FileName)
	if title == "" {
		title = "Document Overview & Key Insights"
	}

	isIndonesian := strings.ToLower(req.Language) != "en"

	if isIndonesian {
		return &GeminiSummaryResult{
			Title:   title,
			Summary: fmt.Sprintf("Dokumen '%s' menyajikan pembahasan komprehensif mengenai konsep-konsep kunci dan penerapannya. Materi ini dirancang untuk memudahkan pemahaman peserta didik melalui struktur pembahasan yang sistematis dan terarah.\n\nPoin-poin utama dalam dokumen ini mencakup pengantar dasar, mekanisme operasional, serta studi kasus praktis yang relevan dengan pembelajaran modern.", req.FileName),
			KeyPoints: []string{
				"Pemahaman fundamental tentang konsep inti yang dipaparkan dalam materi pembelajaran.",
				"Analisis mendalam mengenai metodologi dan langkah implementasi praktis.",
				"Identifikasi manfaat utama serta solusi atas tantangan yang dihadapi.",
				"Rangkuman evaluasi hasil dan rekomendasi penerapan lebih lanjut.",
			},
			Explanation: fmt.Sprintf("Pembahasan dalam dokumen ini berfokus pada penyederhanaan topik kompleks agar mudah dipahami:\n\n1. Konsep Dasar: Menjelaskan landasan teori dengan bahasa yang intuitif.\n2. Alur Pembelajaran: Menghubungkan setiap bagian materi secara berkesinambungan.\n3. Contoh Nyata: Mengaitkan teori dengan skenario dunia nyata untuk memperkuat pemahaman konseptual.",
			),
			Model:      m.model,
			TokenCount: 380,
		}, nil
	}

	return &GeminiSummaryResult{
		Title:   title,
		Summary: fmt.Sprintf("The uploaded document '%s' provides a thorough overview of fundamental concepts and their practical applications. The material is structured to enhance learning retention through methodical topic breakdowns.\n\nKey themes include introductory principles, core mechanisms, and real-world case studies designed for intuitive understanding.", req.FileName),
		KeyPoints: []string{
			"Fundamental principles establishing a solid conceptual foundation.",
			"In-depth breakdown of operational steps and methodologies.",
			"Key benefits, trade-offs, and strategies to overcome common challenges.",
			"Summary of best practices and recommended next steps.",
		},
		Explanation: fmt.Sprintf("This material breaks down complex ideas into manageable learning components:\n\n1. Core Intuition: Explains key principles using intuitive analogies.\n2. Step-by-Step Breakdown: Details how each component connects to the broader subject.\n3. Practical Applications: Connects theoretical knowledge to actionable real-world scenarios.",
		),
		Model:      m.model,
		TokenCount: 380,
	}, nil
}

// ==========================================
// Factory Provider Initializer
// ==========================================

func NewGeminiClient() IGeminiClient {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}

	modelName := os.Getenv("GEMINI_MODEL")
	if modelName == "" {
		modelName = "gemini-3.7-flash"
	}

	if apiKey != "" {
		client, err := NewGenAIClient(apiKey, modelName)
		if err == nil {
			return client
		}
		fmt.Printf("[Gemini] Warning: Failed to initialize live Gemini client (%v). Falling back to mock client.\n", err)
	}

	fmt.Println("[Gemini] No GEMINI_API_KEY or GOOGLE_API_KEY found. Running in simulation/mock mode.")
	return NewMockGeminiClient()
}

// ==========================================
// Helper Functions
// ==========================================

func parseGeminiJSONResponse(text string) (*GeminiSummaryResult, error) {
	clean := strings.TrimSpace(text)

	// Remove markdown code fences if present (e.g. ```json ... ```)
	codeBlockRegex := regexp.MustCompile("(?s)^```(?:json)?\\s*(.*?)\\s*```$")
	if matches := codeBlockRegex.FindStringSubmatch(clean); len(matches) == 2 {
		clean = strings.TrimSpace(matches[1])
	}

	var result GeminiSummaryResult
	if err := json.Unmarshal([]byte(clean), &result); err != nil {
		// Attempt to locate first { and last }
		firstBrace := strings.Index(clean, "{")
		lastBrace := strings.LastIndex(clean, "}")
		if firstBrace >= 0 && lastBrace > firstBrace {
			substr := clean[firstBrace : lastBrace+1]
			if errSub := json.Unmarshal([]byte(substr), &result); errSub == nil {
				return &result, nil
			}
		}
		return nil, err
	}

	return &result, nil
}

func extractTitleFromFilename(filename string) string {
	if filename == "" {
		return ""
	}
	base := filename
	if idx := strings.LastIndex(base, "."); idx != -1 {
		base = base[:idx]
	}
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	return strings.Title(strings.TrimSpace(base))
}
