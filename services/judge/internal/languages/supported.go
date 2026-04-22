package languages

func Supported(language string) bool {
  switch language {
  case "cpp17", "python":
    return true
  default:
    return false
  }
}
