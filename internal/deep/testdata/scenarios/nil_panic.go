// Package handler provides HTTP request handling for the translation API.
package handler

import (
	"encoding/json"
	"net/http"
)

// TranslationRequest represents an incoming translation request.
type TranslationRequest struct {
	SourceText string `json:"source_text"`
	TargetLang string `json:"target_lang"`
	Options    *TranslationOptions
}

// TranslationOptions contains optional translation configuration.
type TranslationOptions struct {
	Formality          string
	Glossary           string
	PreserveFormatting bool
}

// HandleTranslation processes a translation request.
// BUG: panics if req.Options is nil (not provided by client).
func HandleTranslation(w http.ResponseWriter, r *http.Request) {
	var req TranslationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// nil dereference: req.Options may be nil when client omits the field
	if req.Options.Formality == "formal" {
		applyFormalTone(&req)
	}

	result := translateText(req.SourceText, req.TargetLang)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func applyFormalTone(req *TranslationRequest) {
	req.Options.Glossary = "formal-glossary"
}

func translateText(text, lang string) map[string]string {
	return map[string]string{"translation": text, "lang": lang}
}
