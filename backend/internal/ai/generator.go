package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/makosai/backend/internal/models"
)

// Generator interface for AI worksheet generation
type Generator interface {
	GenerateWorksheet(ctx context.Context, input models.WorksheetGeneratorInput) (*models.Worksheet, error)
}

// MockGenerator generates demo worksheets without AI
type MockGenerator struct{}

func NewMockGenerator() *MockGenerator {
	return &MockGenerator{}
}

func (g *MockGenerator) GenerateWorksheet(ctx context.Context, input models.WorksheetGeneratorInput) (*models.Worksheet, error) {
	worksheet := &models.Worksheet{
		ID:                     "ws_" + uuid.New().String()[:8],
		Title:                  fmt.Sprintf("%s Worksheet", input.Topic),
		Subject:                input.Subject,
		Topic:                  input.Topic,
		GradeLevel:             input.GradeLevel,
		Difficulty:             input.Difficulty,
		Language:               input.Language,
		IncludeAnswerKey:       input.IncludeAnswerKey,
		AdditionalInstructions: input.AdditionalInstructions,
		Status:                 "draft",
		Downloads:              0,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	// Generate demo questions
	questions := make([]models.Question, input.QuestionCount)
	for i := 0; i < input.QuestionCount; i++ {
		qType := "multiple_choice"
		if len(input.QuestionTypes) > 0 {
			qType = input.QuestionTypes[i%len(input.QuestionTypes)]
		}
		questions[i] = generateDemoQuestion(i+1, qType, input.Topic)
	}
	worksheet.Questions = questions

	return worksheet, nil
}

func generateDemoQuestion(num int, qType string, topic string) models.Question {
	q := models.Question{
		ID:     fmt.Sprintf("q_%d", num),
		Type:   qType,
		Points: 2,
	}

	switch qType {
	case "multiple_choice":
		q.Question = fmt.Sprintf("Question %d: Which of the following best describes %s?", num, topic)
		q.Options = []string{"Option A - Correct answer", "Option B", "Option C", "Option D"}
		q.CorrectAnswer = "Option A - Correct answer"
		q.Explanation = "This is the correct answer based on the topic."
	case "true_false":
		q.Question = fmt.Sprintf("Question %d: True or False: %s is an important concept to learn.", num, topic)
		q.Options = []string{"True", "False"}
		q.CorrectAnswer = "True"
		q.Explanation = "This statement is true because of its educational significance."
	case "fill_blank":
		q.Question = fmt.Sprintf("Question %d: The main concept of %s is called __________.", num, topic)
		q.CorrectAnswer = "answer"
		q.Explanation = "Fill in the blank with the appropriate term."
	case "short_answer":
		q.Question = fmt.Sprintf("Question %d: Briefly explain the importance of %s.", num, topic)
		q.CorrectAnswer = "A comprehensive answer explaining the importance..."
		q.Explanation = "A good answer should include key concepts."
		q.Points = 5
	case "essay":
		q.Question = fmt.Sprintf("Question %d: Write a detailed essay about %s and its applications.", num, topic)
		q.CorrectAnswer = "Essays are evaluated based on content, structure, and clarity."
		q.Explanation = "Include an introduction, body paragraphs, and conclusion."
		q.Points = 10
	case "matching":
		q.Question = fmt.Sprintf("Question %d: Match the following terms related to %s:", num, topic)
		q.Options = []string{"Term A → Definition 1", "Term B → Definition 2", "Term C → Definition 3"}
		q.CorrectAnswer = []string{"A-1", "B-2", "C-3"}
		q.Explanation = "Match each term with its correct definition."
		q.Points = 3
	default:
		q.Question = fmt.Sprintf("Question %d: Answer the following about %s.", num, topic)
		q.CorrectAnswer = "Sample answer"
	}

	return q
}

// AnthropicGenerator uses Claude API for AI generation
type AnthropicGenerator struct {
	apiKey string
	client *http.Client
}

func NewAnthropicGenerator(apiKey string) *AnthropicGenerator {
	return &AnthropicGenerator{
		apiKey: apiKey,
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

// System prompt for Claude - ensures quality and accuracy
const systemPrompt = `You are an expert educational content creator and curriculum specialist with deep knowledge across all academic subjects. Your role is to create high-quality, pedagogically sound worksheets for students.

CRITICAL REQUIREMENTS:

1. ACCURACY IS PARAMOUNT:
   - Every question MUST have a factually correct answer
   - Double-check all facts, dates, formulas, and scientific information
   - For math problems: solve each problem yourself and verify the answer is correct
   - For science: ensure all scientific facts are accurate and up-to-date
   - For history: verify dates, names, and events
   - For language: ensure grammar and spelling are perfect

2. ANSWER VERIFICATION PROCESS:
   - After creating each question, mentally solve/answer it
   - Verify the correct_answer field matches your solution
   - For multiple choice: ensure exactly ONE option is correct
   - For true/false: verify the statement's truthfulness
   - For fill-in-blank: ensure the answer logically completes the sentence
   - For math: show your work mentally and confirm the numerical answer

3. QUALITY STANDARDS:
   - Questions should be clear, unambiguous, and age-appropriate
   - Avoid trick questions unless specifically requested
   - Explanations should help students understand WHY the answer is correct
   - Distractors (wrong options) should be plausible but clearly incorrect

4. EDUCATIONAL VALUE:
   - Align with curriculum standards for the specified grade level
   - Progress from easier to harder questions when appropriate
   - Include a mix of recall, comprehension, and application questions
   - Make content engaging and relevant to students

5. OUTPUT FORMAT:
   - Always output valid JSON only, no markdown or extra text
   - Follow the exact structure requested
   - Ensure all required fields are present

6. MATHEMATICAL NOTATION (LaTeX):
   - For math questions, use LaTeX notation within the question text
   - Wrap inline math with $...$ (e.g., $x^2 + y^2 = z^2$)
   - Wrap display math with $$...$$ (e.g., $$\frac{a}{b} = c$$)
   - For geometry/diagrams, include a "latex_diagram" field with TikZ code
   - TikZ example: "\\begin{tikzpicture}\\draw (0,0) -- (2,0) -- (1,1.7) -- cycle;\\end{tikzpicture}"
   - Use LaTeX for: fractions, exponents, roots, integrals, summations, matrices
   - Keep diagrams simple and educational`

func (g *AnthropicGenerator) GenerateWorksheet(ctx context.Context, input models.WorksheetGeneratorInput) (*models.Worksheet, error) {
	prompt := g.buildPrompt(input)

	requestBody := map[string]interface{}{
		"model":      "claude-sonnet-4-5-20250929",
		"max_tokens": 4096,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", g.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	// Extract JSON from response
	responseText := apiResp.Content[0].Text
	log.Printf("📝 AI Response (first 500 chars): %.500s", responseText)

	jsonStr := extractJSON(responseText)
	if jsonStr == "" {
		log.Printf("❌ Full response that failed parsing: %s", responseText)
		return nil, fmt.Errorf("no JSON found in response")
	}

	// Parse generated worksheet
	var generated struct {
		Title     string            `json:"title"`
		Questions []models.Question `json:"questions"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &generated); err != nil {
		return nil, fmt.Errorf("failed to parse generated worksheet: %w", err)
	}

	// Build worksheet
	worksheet := &models.Worksheet{
		ID:                     "ws_" + uuid.New().String()[:8],
		Title:                  generated.Title,
		Subject:                input.Subject,
		Topic:                  input.Topic,
		GradeLevel:             input.GradeLevel,
		Difficulty:             input.Difficulty,
		Language:               input.Language,
		Questions:              generated.Questions,
		IncludeAnswerKey:       input.IncludeAnswerKey,
		AdditionalInstructions: input.AdditionalInstructions,
		Status:                 "draft",
		Downloads:              0,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	// Assign IDs if missing
	for i := range worksheet.Questions {
		if worksheet.Questions[i].ID == "" {
			worksheet.Questions[i].ID = fmt.Sprintf("q_%d", i+1)
		}
		if worksheet.Questions[i].Points == 0 {
			worksheet.Questions[i].Points = getDefaultPoints(worksheet.Questions[i].Type)
		}
	}

	// Add SVG diagrams for geometry/physics/circuit topics (FIRST priority)
	if needsDiagrams(input.Subject, input.Topic) {
		log.Println("📐 Adding SVG diagrams for geometry/physics questions...")
		worksheet.Questions = g.addDiagramsIfNeeded(worksheet.Questions, input.Topic)
	}

	// Add images for kindergarten/early grades (only if no SVG was added)
	if isEarlyGrade(input.GradeLevel) && !needsDiagrams(input.Subject, input.Topic) {
		log.Println("🖼️ Adding images for early grade worksheet...")
		for i := range worksheet.Questions {
			if worksheet.Questions[i].Image == "" {
				imageURL := GetImageForQuestion(input.Topic, worksheet.Questions[i].Question)
				if imageURL != "" {
					worksheet.Questions[i].Image = imageURL
					log.Printf("   ✅ Added image for question %d", i+1)
				}
			}
		}
	}

	// Double-check answers for accuracy
	log.Println("🔍 Double-checking answers for accuracy...")
	worksheet.Questions = g.verifyAnswers(ctx, worksheet.Questions, input.Subject, input.Topic)

	return worksheet, nil
}

// verifyAnswers sends questions to AI for answer verification
func (g *AnthropicGenerator) verifyAnswers(ctx context.Context, questions []models.Question, subject, topic string) []models.Question {
	// Build verification prompt
	questionsJSON, err := json.Marshal(questions)
	if err != nil {
		log.Printf("⚠️ Failed to marshal questions for verification: %v", err)
		return questions
	}

	verifyPrompt := fmt.Sprintf(`You are an expert fact-checker and educator. Review these questions and their answers for accuracy.

SUBJECT: %s
TOPIC: %s

QUESTIONS TO VERIFY:
%s

TASK:
1. Check each question's correct_answer for factual accuracy
2. For math problems: solve them yourself and verify the answer
3. For science/history: verify facts are correct
4. If an answer is WRONG, fix it with the correct answer
5. Return the corrected questions array in the same JSON format

IMPORTANT:
- Only output the JSON array of questions
- Keep the exact same structure
- Only change correct_answer and explanation if there's an error
- If all answers are correct, return them unchanged

Output ONLY valid JSON array, no markdown or extra text.`, subject, topic, string(questionsJSON))

	requestBody := map[string]interface{}{
		"model":      "claude-sonnet-4-5-20250929",
		"max_tokens": 4096,
		"messages": []map[string]string{
			{"role": "user", "content": verifyPrompt},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("⚠️ Failed to create verification request: %v", err)
		return questions
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("⚠️ Failed to create verification HTTP request: %v", err)
		return questions
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", g.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := g.client.Do(req)
	if err != nil {
		log.Printf("⚠️ Verification request failed: %v", err)
		return questions
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("⚠️ Verification API error: status %d", resp.StatusCode)
		return questions
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("⚠️ Failed to read verification response: %v", err)
		return questions
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		log.Printf("⚠️ Failed to parse verification response: %v", err)
		return questions
	}

	if len(apiResp.Content) == 0 {
		log.Printf("⚠️ Empty verification response")
		return questions
	}

	// Extract JSON from response
	responseText := apiResp.Content[0].Text
	jsonStr := extractJSON(responseText)
	if jsonStr == "" {
		log.Printf("⚠️ No JSON found in verification response")
		return questions
	}

	// Parse verified questions
	var verifiedQuestions []models.Question
	if err := json.Unmarshal([]byte(jsonStr), &verifiedQuestions); err != nil {
		log.Printf("⚠️ Failed to parse verified questions: %v", err)
		return questions
	}

	log.Printf("✅ Answer verification complete - %d questions verified", len(verifiedQuestions))
	return verifiedQuestions
}

// isEarlyGrade checks if the grade level requires images
func isEarlyGrade(gradeLevel string) bool {
	earlyGrades := []string{"k", "kindergarten", "pre-k", "prek", "1", "2", "1st", "2nd"}
	gradeLower := strings.ToLower(strings.TrimSpace(gradeLevel))
	for _, g := range earlyGrades {
		// Use exact match to avoid "10" matching "1"
		if gradeLower == g {
			return true
		}
	}
	return false
}

func (g *AnthropicGenerator) buildPrompt(input models.WorksheetGeneratorInput) string {
	questionTypes := strings.Join(input.QuestionTypes, ", ")
	if questionTypes == "" {
		questionTypes = "multiple_choice"
	}

	additionalInstr := ""
	if input.AdditionalInstructions != "" {
		additionalInstr = fmt.Sprintf("\n\n📝 ADDITIONAL TEACHER INSTRUCTIONS:\n%s", input.AdditionalInstructions)
	}

	languageInstr := "English"
	switch input.Language {
	case "tr":
		languageInstr = "Turkish (Türkçe) - Generate ALL content including questions, options, answers, and explanations in Turkish"
	case "es":
		languageInstr = "Spanish (Español) - Generate ALL content in Spanish"
	case "fr":
		languageInstr = "French (Français) - Generate ALL content in French"
	case "de":
		languageInstr = "German (Deutsch) - Generate ALL content in German"
	case "en":
		languageInstr = "English - Generate all content in English"
	default:
		languageInstr = fmt.Sprintf("%s - Generate ALL content in this language", input.Language)
	}

	return fmt.Sprintf(`Generate an educational worksheet with the following specifications:

📚 WORKSHEET SPECIFICATIONS:
━━━━━━━━━━━━━━━━━━━━━━━━━━
• Topic: %s
• Subject: %s
• Grade Level: %s
• Difficulty: %s
• Number of Questions: %d
• Question Types: %s
• Language: %s%s

🎯 OUTPUT REQUIREMENTS:
━━━━━━━━━━━━━━━━━━━━━━━━━━
Generate a JSON object with this EXACT structure:

{
  "title": "Creative and descriptive worksheet title",
  "questions": [
    {
      "id": "q_1",
      "type": "multiple_choice",
      "question": "Question text with $LaTeX$ math notation if needed",
      "options": ["Option A with $math$", "Option B", "Option C", "Option D"],
      "correct_answer": "The correct option (must match exactly one of the options)",
      "explanation": "Educational explanation with $math$ if needed",
      "points": 10
    }
  ]
}

📐 MATHEMATICAL NOTATION:
━━━━━━━━━━━━━━━━━━━━━━━━━━
- Use LaTeX for ALL mathematical expressions
- Inline math: $x^2$, $\frac{1}{2}$, $\sqrt{16}$
- Display math: $$\sum_{i=1}^{n} i = \frac{n(n+1)}{2}$$
- Fractions: $\frac{a}{b}$
- Exponents: $x^2$, $2^{10}$
- Roots: $\sqrt{x}$, $\sqrt[3]{8}$
- Greek letters: $\pi$, $\theta$, $\alpha$
- Geometry: $\angle ABC$, $\triangle ABC$, $\perp$, $\parallel$
- For early grades: Keep it simple - use LaTeX only for basic operations

📋 QUESTION TYPE FORMATS:
━━━━━━━━━━━━━━━━━━━━━━━━━━
• multiple_choice: 4 options array, correct_answer = exact option text
• true_false: options = ["True", "False"], correct_answer = "True" or "False"
• fill_blank: use __________ for blank, correct_answer = the word/phrase
• short_answer: no options, correct_answer = sample correct response
• essay: no options, points = higher value, correct_answer = grading criteria
• matching: options = ["Term A → Definition 1", ...], correct_answer = ["A-1", ...]

⚠️ VERIFICATION CHECKLIST (Do this for EACH question):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
□ Is the question factually accurate?
□ Is the correct_answer actually correct? (Solve/verify it yourself)
□ For math: Did you calculate the answer and verify it's right?
□ For science: Is the scientific information accurate?
□ For multiple choice: Is there exactly ONE correct answer?
□ Does the explanation help students understand the concept?

Output ONLY the JSON object. No markdown, no code blocks, no extra text.`,
		input.Topic, input.Subject, input.GradeLevel, input.Difficulty,
		input.QuestionCount, questionTypes, languageInstr, additionalInstr)
}

func extractJSON(text string) string {
	// Try to find JSON block in markdown
	if start := strings.Index(text, "```json"); start != -1 {
		start += 7
		if end := strings.Index(text[start:], "```"); end != -1 {
			return strings.TrimSpace(text[start : start+end])
		}
	}

	// Try to find raw JSON
	if start := strings.Index(text, "{"); start != -1 {
		depth := 0
		for i := start; i < len(text); i++ {
			if text[i] == '{' {
				depth++
			} else if text[i] == '}' {
				depth--
				if depth == 0 {
					return text[start : i+1]
				}
			}
		}
	}

	return ""
}

func getDefaultPoints(qType string) int {
	switch qType {
	case "essay":
		return 10
	case "short_answer":
		return 5
	case "matching":
		return 3
	default:
		return 2
	}
}

// needsDiagrams checks if the subject/topic requires diagrams
func needsDiagrams(subject, topic string) bool {
	topic = strings.ToLower(topic)
	subject = strings.ToLower(subject)

	diagramKeywords := []string{
		"geometry", "triangle", "circle", "angle", "polygon", "area", "perimeter",
		"circuit", "electrical", "resistor", "voltage", "current",
		"physics", "force", "motion", "vector",
		"trigonometry", "sine", "cosine", "tangent",
	}

	for _, keyword := range diagramKeywords {
		if strings.Contains(topic, keyword) || strings.Contains(subject, keyword) {
			return true
		}
	}
	return false
}

// addDiagramsIfNeeded adds SVG diagrams to questions that need them
func (g *AnthropicGenerator) addDiagramsIfNeeded(questions []models.Question, topic string) []models.Question {
	topic = strings.ToLower(topic)

	for i := range questions {
		q := &questions[i]
		questionLower := strings.ToLower(q.Question)

		// Skip if already has an image
		if q.Image != "" {
			continue
		}

		// Triangle questions (including trigonometry problems)
		if strings.Contains(questionLower, "triangle") || strings.Contains(questionLower, "△") ||
			strings.Contains(questionLower, "law of cosines") || strings.Contains(questionLower, "law of sines") ||
			strings.Contains(questionLower, "cosine rule") || strings.Contains(questionLower, "sine rule") ||
			(strings.Contains(questionLower, "sides") && strings.Contains(questionLower, "angle")) {
			q.Image = generateTriangleSVG(q.Question)
			log.Printf("   ✅ Added triangle SVG for question %d", i+1)
		}

		// Circle questions
		if strings.Contains(questionLower, "circle") || strings.Contains(questionLower, "radius") {
			q.Image = generateCircleSVG(q.Question)
			log.Printf("   ✅ Added circle SVG for question %d", i+1)
		}

		// Circuit questions
		if strings.Contains(questionLower, "circuit") || strings.Contains(questionLower, "resistor") {
			q.Image = generateCircuitSVG(q.Question)
			log.Printf("   ✅ Added circuit SVG for question %d", i+1)
		}
	}

	return questions
}

// generateTriangleSVG creates an SVG for triangle problems
func generateTriangleSVG(question string) string {
	// Extract values from question if possible
	a, b, c := "a", "b", "c"

	// Try to extract side lengths
	re := regexp.MustCompile(`[abc]\s*=\s*(\d+)`)
	matches := re.FindAllStringSubmatch(question, -1)
	if len(matches) >= 3 {
		a = matches[0][1]
		b = matches[1][1]
		c = matches[2][1]
	}

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 180" width="200" height="180">
  <polygon points="100,20 30,160 170,160" fill="none" stroke="#0d9488" stroke-width="2.5"/>
  <text x="100" y="12" text-anchor="middle" font-size="14" font-weight="bold" fill="#1e293b">A</text>
  <text x="20" y="175" text-anchor="middle" font-size="14" font-weight="bold" fill="#1e293b">B</text>
  <text x="180" y="175" text-anchor="middle" font-size="14" font-weight="bold" fill="#1e293b">C</text>
  <text x="55" y="85" text-anchor="middle" font-size="13" fill="#0f766e">%s</text>
  <text x="145" y="85" text-anchor="middle" font-size="13" fill="#0f766e">%s</text>
  <text x="100" y="178" text-anchor="middle" font-size="13" fill="#0f766e">%s</text>
</svg>`, a, b, c)
}

// generateCircleSVG creates an SVG for circle problems
func generateCircleSVG(question string) string {
	radius := "r"

	// Try to extract radius
	re := regexp.MustCompile(`radius\s*(?:of|is|=)?\s*(\d+)`)
	if match := re.FindStringSubmatch(strings.ToLower(question)); len(match) > 1 {
		radius = match[1]
	}

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 200" width="200" height="200">
  <circle cx="100" cy="100" r="70" fill="none" stroke="#0d9488" stroke-width="2.5"/>
  <circle cx="100" cy="100" r="3" fill="#0d9488"/>
  <line x1="100" y1="100" x2="170" y2="100" stroke="#f97316" stroke-width="2" stroke-dasharray="5,3"/>
  <text x="100" y="95" text-anchor="middle" font-size="12" fill="#1e293b">O</text>
  <text x="135" y="95" text-anchor="middle" font-size="13" font-weight="bold" fill="#f97316">r = %s</text>
</svg>`, radius)
}

// generateCircuitSVG creates an SVG for circuit problems
func generateCircuitSVG(question string) string {
	return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 240 120" width="240" height="120">
  <rect x="20" y="30" width="200" height="60" fill="none" stroke="#0d9488" stroke-width="2"/>
  <rect x="80" y="25" width="30" height="10" fill="#f97316" stroke="#ea580c" stroke-width="1"/>
  <text x="95" y="20" text-anchor="middle" font-size="10" fill="#1e293b">R₁</text>
  <rect x="130" y="25" width="30" height="10" fill="#f97316" stroke="#ea580c" stroke-width="1"/>
  <text x="145" y="20" text-anchor="middle" font-size="10" fill="#1e293b">R₂</text>
  <text x="30" y="65" font-size="14" fill="#1e293b">+</text>
  <text x="200" y="65" font-size="14" fill="#1e293b">−</text>
  <text x="120" y="110" text-anchor="middle" font-size="11" fill="#64748b">Series Circuit</text>
</svg>`
}

// OpenAIGenerator uses OpenAI API (placeholder)
type OpenAIGenerator struct {
	apiKey string
}

func NewOpenAIGenerator(apiKey string) *OpenAIGenerator {
	return &OpenAIGenerator{apiKey: apiKey}
}

func (g *OpenAIGenerator) GenerateWorksheet(ctx context.Context, input models.WorksheetGeneratorInput) (*models.Worksheet, error) {
	// Fallback to mock for now
	mock := NewMockGenerator()
	return mock.GenerateWorksheet(ctx, input)
}
