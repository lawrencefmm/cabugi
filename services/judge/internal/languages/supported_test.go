package languages

import "testing"

func TestSupported(t *testing.T) {
  cases := []struct {
    name     string
    language string
    want     bool
  }{
    {name: "cpp17 supported", language: "cpp17", want: true},
    {name: "python supported", language: "python", want: true},
    {name: "unsupported language rejected", language: "java", want: false},
  }

  for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) {
      if got := Supported(tc.language); got != tc.want {
        t.Fatalf("Supported(%q) = %t, want %t", tc.language, got, tc.want)
      }
    })
  }
}
