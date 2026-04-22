package submissions

type Language string

const (
	LanguageCPP17  Language = "cpp17"
	LanguagePython Language = "python"
)

func SupportedLanguage(language string) bool {
	switch Language(language) {
	case LanguageCPP17, LanguagePython:
		return true
	default:
		return false
	}
}
