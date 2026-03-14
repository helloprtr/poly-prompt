package translate

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode"
)

const (
	DecisionTranslated      = "translated"
	DecisionSkipped         = "skipped"
	DecisionPartialPreserve = "partial-preserve"
)

type Mode string

const (
	ModeAuto  Mode = "auto"
	ModeForce Mode = "force"
	ModeSkip  Mode = "skip"
)

type Outcome struct {
	Text       string
	Decision   string
	SourceLang string
	TargetLang string
}

var preservePatterns = []*regexp.Regexp{
	regexp.MustCompile("(?s)```.*?```"),
	regexp.MustCompile("`[^`\\n]+`"),
	regexp.MustCompile(`https?://\S+`),
	regexp.MustCompile(`\$\{?[A-Z0-9_]+\}?`),
	regexp.MustCompile(`(?:[A-Za-z]:\\[^\s]+|(?:\./|\../|/)[^\s]+)`),
	regexp.MustCompile(`(?m)^\s*(?:at .+|File ".+", line \d+.*|Traceback .+|panic: .+)\s*$`),
}

func ApplyPolicy(ctx context.Context, translator Translator, req Request, mode Mode) (Outcome, error) {
	sourceLang := strings.ToLower(strings.TrimSpace(req.SourceLang))
	if sourceLang == "" {
		sourceLang = "auto"
	}
	targetLang := strings.ToLower(strings.TrimSpace(req.TargetLang))
	if targetLang == "" {
		targetLang = "en"
	}

	outcome := Outcome{
		SourceLang: sourceLang,
		TargetLang: targetLang,
	}

	if mode == "" {
		mode = ModeAuto
	}
	if mode == ModeSkip {
		outcome.Text = req.Text
		outcome.Decision = DecisionSkipped
		return outcome, nil
	}

	if mode == ModeAuto && shouldSkipTranslation(req.Text, sourceLang, targetLang) {
		outcome.Text = req.Text
		outcome.Decision = DecisionSkipped
		return outcome, nil
	}

	protectedText, restore, preserved := protectSegments(req.Text)
	if translator == nil {
		return Outcome{}, errors.New("translator is not configured")
	}
	translated, err := translator.Translate(ctx, Request{
		Text:       protectedText,
		SourceLang: sourceLang,
		TargetLang: targetLang,
	})
	if err != nil {
		return Outcome{}, err
	}

	for token, original := range restore {
		translated = strings.ReplaceAll(translated, token, original)
	}

	outcome.Text = translated
	outcome.Decision = DecisionTranslated
	if preserved {
		outcome.Decision = DecisionPartialPreserve
	}

	return outcome, nil
}

func shouldSkipTranslation(text, sourceLang, targetLang string) bool {
	if strings.TrimSpace(text) == "" {
		return true
	}
	if sourceLang != "" && sourceLang != "auto" && normalizeComparableLang(sourceLang) == normalizeComparableLang(targetLang) {
		return true
	}
	if !strings.HasPrefix(normalizeComparableLang(targetLang), "en") {
		return false
	}

	letters := 0
	asciiLetters := 0
	nonASCII := 0
	for _, r := range text {
		if unicode.IsLetter(r) {
			letters++
			if r <= unicode.MaxASCII {
				asciiLetters++
			}
		}
		if r > unicode.MaxASCII && !unicode.IsSpace(r) {
			nonASCII++
		}
	}

	if letters == 0 {
		return false
	}
	if nonASCII == 0 {
		return true
	}

	return float64(asciiLetters)/float64(letters) >= 0.85
}

func protectSegments(text string) (string, map[string]string, bool) {
	restore := map[string]string{}
	protected := text
	index := 0

	for _, pattern := range preservePatterns {
		protected = pattern.ReplaceAllStringFunc(protected, func(match string) string {
			token := preserveToken(index)
			restore[token] = match
			index++
			return token
		})
	}

	return protected, restore, len(restore) > 0
}

func preserveToken(index int) string {
	return "PRTRPRESERVE_" + strconvItoa(index) + "_TOKEN"
}

func normalizeComparableLang(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "en-us", "en-gb":
		return "en"
	default:
		return value
	}
}

func strconvItoa(value int) string {
	if value == 0 {
		return "0"
	}

	var digits [20]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + (value % 10))
		value /= 10
	}
	return string(digits[i:])
}
