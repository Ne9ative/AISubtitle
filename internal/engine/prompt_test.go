package engine

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildPromptContainsEssentials(t *testing.T) {
	p := buildPrompt([]string{"Hello", "World"}, []string{"Prev line"}, "ANGLAIS")
	for _, want := range []string{"ANGLAIS", "FRANÇAIS", "Hello", "World", "Prev line", "translations"} {
		if !strings.Contains(p, want) {
			t.Fatalf("le prompt ne contient pas %q\n---\n%s", want, p)
		}
	}
}

func TestParseTranslationsValid(t *testing.T) {
	got, err := parseTranslations(`{"translations":["a","b"]}`, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("got %v", got)
	}
}

func TestParseTranslationsMarkdownFence(t *testing.T) {
	got, err := parseTranslations("```json\n{\"translations\": [\"x\"]}\n```", 1)
	if err != nil || !reflect.DeepEqual(got, []string{"x"}) {
		t.Fatalf("got %v err %v", got, err)
	}
}

func TestParseTranslationsSurroundingProse(t *testing.T) {
	got, err := parseTranslations(`Voici la traduction : {"translations":["y"]} — voilà.`, 1)
	if err != nil || !reflect.DeepEqual(got, []string{"y"}) {
		t.Fatalf("got %v err %v", got, err)
	}
}

func TestParseTranslationsWrongCount(t *testing.T) {
	if _, err := parseTranslations(`{"translations":["a"]}`, 2); err == nil {
		t.Fatal("attendu une erreur de comptage")
	}
}

func TestParseTranslationsMalformed(t *testing.T) {
	if _, err := parseTranslations(`{"translations": [`, 1); err == nil {
		t.Fatal("attendu une erreur JSON")
	}
}

func TestParseTranslationsNoJSON(t *testing.T) {
	if _, err := parseTranslations(`pas de json ici`, 1); err == nil {
		t.Fatal("attendu une erreur (aucun JSON)")
	}
}
