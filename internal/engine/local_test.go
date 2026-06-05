package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLocalTranslateOverHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/v1/chat/completions") {
			t.Errorf("chemin inattendu: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"translations\":[\"Salut\"]}"}}]}`))
	}))
	defer srv.Close()

	l := &Local{baseURL: srv.URL, client: &http.Client{Timeout: 5 * time.Second}}
	got, err := l.Translate(context.Background(), []string{"Hi"}, nil, "ANGLAIS")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"Salut"}) {
		t.Fatalf("got %v", got)
	}
}
