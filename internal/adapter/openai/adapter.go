package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"lexbot/internal/service"

	"github.com/sashabaranov/go-openai"
)

const systemPrompt = `
You are an assistant specialized in teaching English to Brazilian Portuguese speakers.
Return ONLY a valid JSON object, no markdown, no text before or after.

Word received: "{{WORD}}"

Exact format:
{
  "word": "the word exactly as received",
  "translation": "main translation in Brazilian Portuguese",
  "grammar_class": "noun | verb | adjective | adverb | conjunction | preposition | other",
  "phonetic": "IPA transcription",
  "definition_en": "short definition in English, without using the word itself",
  "example_en": "natural English sentence at B1-B2 level",
  "example_pt": "Brazilian Portuguese translation of the sentence",
  "synonyms": ["synonym1", "synonym2", "synonym3"],
  "quiz_tip": "short hint in Brazilian Portuguese that helps recall the word without giving it away (e.g. grammar class + semantic or morphological clue)",
  "quiz_error_explain": "short explanation in Brazilian Portuguese of the word's meaning and correct usage, written for someone who just got it wrong. Mention common traps or confusions with similar words if relevant.",
  "connected_speech": {
    "example_phrase": "a complete, natural English sentence containing the word, at least 5 words long, where linking occurs somewhere in the phrase",
    "example_phrase_pt": "Brazilian Portuguese translation of the phrase",
    "linking_words": "a single string with the exact two words that link together, separated by a space. Ex: \"turn it\" or \"pick up\". Never return an array.",
    "connected_sound": "how ONLY the two linking words sound when fused together. Use only plain letters and Portuguese accents, no phonetic symbols. Ex: 'turn it' → 'tôr-nit', 'pick it' → 'pí-kit'",
    "connected_sound_pt": "how ONLY the two fused linking words would sound to a Brazilian reader. Use ONLY Portuguese letters and accents (á, ê, ô, etc). NO IPA symbols allowed. Ex: 'turn it' → 'tôr-nit', 'not at' → 'nô-tat'",
    "tip_pt": "explanation in Brazilian Portuguese covering: (1) which final sound of the FIRST linking word merges with which initial sound of the SECOND linking word, (2) other frequent contexts where a Brazilian will hear this same linking pattern"
  },
  "quiz": {
    "multiple_choice": {
      "question": "sentence IN ENGLISH with the word replaced by a single ___",
      "correct": "the received word — always a SINGLE WORD, never a phrase",
      "distractors": ["single english word", "single english word", "single english word"]
    },
    "complete_sentence": {
      "question": "different sentence IN ENGLISH with a single ___ in place of the word",
      "correct": "the received word — always a SINGLE WORD, never a phrase"
    },
    "reverse": {
      "question": "a clear, direct description in Brazilian Portuguese that leads the user to guess the word. Do not mix context or storytelling into the question — ask directly.",
      "correct": "the received word, in English"
    }
  }
}

GENERAL RULES:
- Distractors must be plausible: same grammar class, similar meaning, but wrong in context
- The registered word NEVER appears as a distractor
- Never use the word itself in the "reverse" question or in "quiz_tip"
- Quiz sentences must differ from each other and from example_en
- quiz_tip must not reveal the answer — only guide toward it
- quiz_error_explain: maximum 2 sentences, direct and didactic, in Brazilian Portuguese
- quiz: all questions and distractors in English. Only the "reverse" question must be in Brazilian Portuguese
- quiz.multiple_choice.correct and quiz.complete_sentence.correct must ALWAYS be a single word — the exact registered word. Never return a phrase, collocation or multiple words in these fields. The ___ gap represents exactly one word.
- Before finalizing quiz.multiple_choice and quiz.complete_sentence, mentally test: "does [correct] fit as a single word, grammatically and naturally, in place of ___?" — if not, rewrite the sentence
- For functional words (adverbs, conjunctions, prepositions) like "instead", "although", "rather": build sentences where the word appears at the end of a clause or after the main verb, so it fits naturally as a single word in the gap. Examples for "instead": "She didn't want coffee, so she had tea ___." / "He was tired, so he stayed home ___."
- All fields inside connected_speech are required. Never omit any of them, especially tip_pt

CONNECTED SPEECH RULES — READ CAREFULLY:
The only phenomenon trained here is LINKING.
Linking occurs ONLY when the preceding word ends in a CONSONANT sound and the following word begins with a VOWEL sound. The two sounds merge together.

Before choosing the two linking words, explicitly verify:
1. What is the FINAL sound of the first word? Is it a consonant sound?
2. What is the INITIAL sound of the second word? Is it a vowel sound?
3. Only if BOTH conditions are true, it is valid linking.

CORRECT linking examples:
- "turn it"    → /n/ final + /ɪ/ initial = consonant + vowel ✓ → "tôr-nit"
- "pick up"    → /k/ final + /ʌ/ initial = consonant + vowel ✓ → "pí-kap"
- "an apple"   → /n/ final + /æ/ initial = consonant + vowel ✓ → "a-næpl"
- "not at all" → /t/ final + /æ/ initial = consonant + vowel ✓ → "nô-ta-tôl"
- "come over"  → /m/ final + /oʊ/ initial = consonant + vowel ✓ → "câ-môvér"
- "of him"     → /v/ final + /ɪ/ initial = consonant + vowel ✓ → "ô-vim"

EXAMPLES THAT ARE NOT LINKING — never use these patterns:
- "go instead" → /oʊ/ final + /ɪ/ initial = vowel + vowel ✗
- "very easy"  → /i/ final + /i/ initial = vowel + vowel ✗
- "of my"      → /v/ final + /m/ initial = consonant + consonant ✗
- "and the"    → /d/ final + /ð/ initial = consonant + consonant ✗
- "stay in"    → /eɪ/ final + /ɪ/ initial = vowel + vowel ✗

HIERARCHY FOR CHOOSING the two linking words:
1. Prioritize linking directly involving the registered word
   (word ends in consonant + next word begins with vowel,
   or previous word ends in consonant + registered word begins with vowel)
2. If not possible, build a natural sentence where linking occurs
   between two other words in the phrase, and indicate which ones in linking_words

RULES FOR connected_sound AND connected_sound_pt:
- Both fields represent ONLY the two fused linking words — never the full sentence
- NO IPA symbols allowed in either field: ə ɪ ː ɛ ɾ ʃ ʒ θ ð æ ʌ ɔ ʊ ɑ ŋ
- Use only plain letters and Portuguese accents
- The result must be readable by any Brazilian with no knowledge of phonetics
- tip_pt is mandatory — never omit it
- tip_pt must be factually correct: only describe the final sound of the first linking word as a consonant if it truly is one — verify before writing`

// cleanJSONMarkdown removes Markdown code block formatting to ensure valid JSON parsing
func cleanJSONMarkdown(raw string) string {
	raw = strings.TrimSpace(raw)
	// Match ```json ... ``` or just ``` ... ```
	re := regexp.MustCompile("(?s)^```(?:json)?\n?(.*?)\n?```$")
	match := re.FindStringSubmatch(raw)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return raw
}

// Adapter implements the service.AIProvider interface using the OpenAI Go SDK
type Adapter struct {
	client *openai.Client
}

// NewAdapter creates a new OpenAI adapter (configured for Groq or standard OpenAI)
func NewAdapter(apiKey string, baseURL string) *Adapter {
	clientConfig := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		clientConfig.BaseURL = baseURL
	}
	client := openai.NewClientWithConfig(clientConfig)
	return &Adapter{client: client}
}

// ProcessWord queries the AI and returns a structured Word object
func (a *Adapter) ProcessWord(ctx context.Context, word string) (*service.Word, error) {
	resp, err := a.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: "llama-3.1-8b-instant", // specific to Groq, if you want it configurable, pass it as well
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: word,
				},
			},
			Temperature: 0.2, // Low temperature for more deterministic JSON
		},
	)

	if err != nil {
		return nil, fmt.Errorf("ChatCompletion error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	rawJSON := resp.Choices[0].Message.Content
	cleanJSON := cleanJSONMarkdown(rawJSON)

	var aiResponse service.Word
	err = json.Unmarshal([]byte(cleanJSON), &aiResponse)
	if err != nil {
		return nil, fmt.Errorf("JSON Unmarshal error: %w\nRAW: %s", err, rawJSON)
	}

	return &aiResponse, nil
}
