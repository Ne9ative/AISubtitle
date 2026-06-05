package subs

import (
	"reflect"
	"testing"
)

func TestMakeBatchesEmpty(t *testing.T) {
	if got := MakeBatches(nil, 5, 2); len(got) != 0 {
		t.Fatalf("attendu 0 lot, got %d", len(got))
	}
}

func TestMakeBatchesCoversAllOnce(t *testing.T) {
	texts := []string{"a", "b", "c", "d", "e"}
	var got []string
	for _, b := range MakeBatches(texts, 2, 1) {
		got = append(got, b.Lines...)
	}
	if !reflect.DeepEqual(got, texts) {
		t.Fatalf("les lots ne couvrent pas tout: %v", got)
	}
}

func TestMakeBatchesContext(t *testing.T) {
	texts := []string{"a", "b", "c", "d", "e"}
	batches := MakeBatches(texts, 2, 1)
	if len(batches) != 3 {
		t.Fatalf("attendu 3 lots, got %d", len(batches))
	}
	if len(batches[0].Context) != 0 {
		t.Fatalf("lot 0 ne devrait pas avoir de contexte: %v", batches[0].Context)
	}
	if !reflect.DeepEqual(batches[1].Context, []string{"b"}) {
		t.Fatalf("contexte lot 1 = %v, attendu [b]", batches[1].Context)
	}
	if batches[1].Start != 2 {
		t.Fatalf("Start lot 1 = %d, attendu 2", batches[1].Start)
	}
	if !reflect.DeepEqual(batches[2].Lines, []string{"e"}) {
		t.Fatalf("lignes lot 2 = %v, attendu [e]", batches[2].Lines)
	}
	if !reflect.DeepEqual(batches[2].Context, []string{"d"}) {
		t.Fatalf("contexte lot 2 = %v, attendu [d]", batches[2].Context)
	}
}

func TestMakeBatchesBigBatch(t *testing.T) {
	batches := MakeBatches([]string{"a", "b", "c"}, 10, 2)
	if len(batches) != 1 || len(batches[0].Lines) != 3 {
		t.Fatalf("attendu 1 lot de 3 lignes, got %d lots", len(batches))
	}
}

func TestMakeBatchesContextSpansMultiple(t *testing.T) {
	batches := MakeBatches([]string{"a", "b", "c", "d", "e", "f"}, 2, 2)
	if !reflect.DeepEqual(batches[1].Context, []string{"a", "b"}) {
		t.Fatalf("contexte lot 1 = %v, attendu [a b]", batches[1].Context)
	}
}
