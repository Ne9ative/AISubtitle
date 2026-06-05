package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestGeminiTranslate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, ":generateContent") {
			t.Errorf("chemin inattendu: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"translations\":[\"Bonjour\",\"Monde\"]}"}]}}]}`))
	}))
	defer srv.Close()

	g := NewGemini("fakekey", "gemini-2.0-flash")
	g.baseURL = srv.URL
	got, err := g.Translate(context.Background(), []string{"Hello", "World"}, nil, "ANGLAIS")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"Bonjour", "Monde"}) {
		t.Fatalf("got %v", got)
	}
}

func TestGeminiHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "quota exceeded", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	g := NewGemini("k", "m")
	g.baseURL = srv.URL
	if _, err := g.Translate(context.Background(), []string{"Hi"}, nil, "ANGLAIS"); err == nil {
		t.Fatal("attendu une erreur HTTP")
	}
}
