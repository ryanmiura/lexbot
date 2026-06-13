package service

type Word struct {
	Word             string          `json:"word"`
	Translation      string          `json:"translation"`
	GrammarClass     string          `json:"grammar_class"`
	Phonetic         string          `json:"phonetic"`
	DefinitionEN     string          `json:"definition_en"`
	ExampleEN        string          `json:"example_en"`
	ExamplePT        string          `json:"example_pt"`
	Synonyms         []string        `json:"synonyms"`
	QuizTip          string          `json:"quiz_tip"`
	QuizErrorExplain string          `json:"quiz_error_explain"`
	ConnectedSpeech  ConnectedSpeech `json:"connected_speech"`
	Quiz             Quiz            `json:"quiz"`
}

type ConnectedSpeech struct {
	ExamplePhrase    string `json:"example_phrase"`
	ExamplePhrasePT  string `json:"example_phrase_pt"`
	LinkingWords     string `json:"linking_words"`
	ConnectedSound   string `json:"connected_sound"`
	ConnectedSoundPT string `json:"connected_sound_pt"`
	TipPT            string `json:"tip_pt"`
}

type Quiz struct {
	MultipleChoice   QuizQuestion `json:"multiple_choice"`
	CompleteSentence QuizQuestion `json:"complete_sentence"`
	Reverse          QuizQuestion `json:"reverse"`
}

type QuizQuestion struct {
	Question    string   `json:"question"`
	Correct     string   `json:"correct"`
	Distractors []string `json:"distractors,omitempty"`
}
