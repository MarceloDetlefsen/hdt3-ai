package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	openai "github.com/openai/openai-go/v3"
)

func TestIsByeCommand(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "lower", input: "bye", want: true},
		{name: "mixed", input: "ByE", want: true},
		{name: "upper", input: "BYE", want: true},
		{name: "with spaces", input: "  Bye  ", want: true},
		{name: "different word", input: "adios", want: false},
		{name: "empty", input: "", want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := isByeCommand(tc.input); got != tc.want {
				t.Fatalf("isByeCommand(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestIsEmptyInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "empty", input: "", want: true},
		{name: "spaces", input: "   ", want: true},
		{name: "tabs and newlines", input: "\n\t  ", want: true},
		{name: "word", input: "hola", want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := isEmptyInput(tc.input); got != tc.want {
				t.Fatalf("isEmptyInput(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestLoadFAQs(t *testing.T) {
	t.Parallel()

	t.Run("loads content", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "faqs.txt")
		want := "Pregunta 1\nRespuesta 1"

		if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}

		got, err := loadFAQs(path)
		if err != nil {
			t.Fatalf("loadFAQs() error = %v", err)
		}
		if got != want {
			t.Fatalf("loadFAQs() = %q, want %q", got, want)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "missing.txt")

		_, err := loadFAQs(path)
		if err == nil {
			t.Fatal("loadFAQs() error = nil, want error")
		}
		if !strings.Contains(err.Error(), path) {
			t.Fatalf("loadFAQs() error %q does not include path %q", err.Error(), path)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "empty.txt")

		if err := os.WriteFile(path, []byte("   \n\t"), 0o600); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}

		_, err := loadFAQs(path)
		if err == nil {
			t.Fatal("loadFAQs() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "está vacío") {
			t.Fatalf("loadFAQs() error %q does not mention empty content", err.Error())
		}
	})
}

func TestBuildPrompt(t *testing.T) {
	t.Parallel()

	faqs := "Pregunta 1\nRespuesta 1"
	question := "¿Cuándo y dónde se llevará a cabo el evento?"

	prompt := buildPrompt(faqs, question)
	checks := []string{
		faqs,
		question,
		promptDelimiter,
		promptFAQsLabel,
		promptQuestionLabel,
	}

	for _, check := range checks {
		if !strings.Contains(prompt, check) {
			t.Fatalf("buildPrompt() output does not contain %q: %q", check, prompt)
		}
	}
}

func TestBuildMessages(t *testing.T) {
	t.Parallel()

	msgs := buildMessages("FAQs", "Pregunta")
	if len(msgs) != 2 {
		t.Fatalf("buildMessages() len = %d, want 2", len(msgs))
	}

	if msgs[0].OfDeveloper == nil {
		t.Fatal("developer message is nil")
	}

	devContent, ok := msgs[0].GetContent().AsAny().(*string)
	if !ok || devContent == nil {
		t.Fatalf("developer message content has unexpected type %T", msgs[0].GetContent().AsAny())
	}
	if !strings.Contains(*devContent, "No inventar ni completar datos faltantes.") {
		t.Fatalf("developer message content = %q, want instruction text", *devContent)
	}
	if !strings.Contains(*devContent, "No puedo responder esa pregunta con la información disponible en las FAQs.") {
		t.Fatalf("developer message content = %q, want fallback answer", *devContent)
	}

	if msgs[1].OfUser == nil {
		t.Fatal("user message is nil")
	}

	userContent, ok := msgs[1].GetContent().AsAny().(*string)
	if !ok || userContent == nil {
		t.Fatalf("user message content has unexpected type %T", msgs[1].GetContent().AsAny())
	}
	if !strings.Contains(*userContent, "FAQs") || !strings.Contains(*userContent, "Pregunta") {
		t.Fatalf("user message content = %q, want faqs and question", *userContent)
	}
}

func TestExtractAnswer(t *testing.T) {
	t.Parallel()

	t.Run("nil response", func(t *testing.T) {
		t.Parallel()

		_, err := extractAnswer(nil)
		if err == nil {
			t.Fatal("extractAnswer(nil) error = nil, want error")
		}
	})

	t.Run("empty choices", func(t *testing.T) {
		t.Parallel()

		resp := &openai.ChatCompletion{}
		_, err := extractAnswer(resp)
		if err == nil {
			t.Fatal("extractAnswer() error = nil, want error")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		t.Parallel()

		resp := &openai.ChatCompletion{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{Content: "   "},
				},
			},
		}
		_, err := extractAnswer(resp)
		if err == nil {
			t.Fatal("extractAnswer() error = nil, want error")
		}
	})

	t.Run("valid content", func(t *testing.T) {
		t.Parallel()

		resp := &openai.ChatCompletion{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{Content: "Respuesta útil"},
				},
			},
		}
		got, err := extractAnswer(resp)
		if err != nil {
			t.Fatalf("extractAnswer() error = %v", err)
		}
		if got != "Respuesta útil" {
			t.Fatalf("extractAnswer() = %q, want %q", got, "Respuesta útil")
		}
	})
}
