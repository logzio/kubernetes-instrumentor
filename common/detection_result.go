package common

type DetectionResult struct {
	LanguageByContainer    []LanguageByContainer  `json:"languageByContainer"`
	ApplicationByContainer ApplicationByContainer `json:"applicationByContainer"`
}
