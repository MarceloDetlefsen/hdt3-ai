package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const faqsPath = "data/FAQs_Parachute_SA_Guatemala_2026.txt"

func main() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "error cargando .env: %v\n", err)
		os.Exit(1)
	}

	apiKey := strings.TrimSpace(os.Getenv("GROQ_API_KEY"))
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "GROQ_API_KEY no está configurada. Defínela en .env o expórtala en tu sistema.")
		os.Exit(1)
	}

	faqs, err := loadFAQs(faqsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_ = apiKey
	_ = faqs

	fmt.Println("FAQs cargadas correctamente.")
}

func loadFAQs(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}

	faqs := strings.TrimSpace(string(content))
	if faqs == "" {
		return "", fmt.Errorf("%s está vacío", path)
	}

	return faqs, nil
}
