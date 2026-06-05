package subs

// Batch = un groupe de lignes à traduire, accompagné de quelques lignes
// précédentes en contexte (non traduites, pour aider le modèle).
type Batch struct {
	Context []string // lignes précédentes (contexte, non traduit)
	Lines   []string // lignes à traduire
	Start   int      // index de la 1re ligne dans le tableau source
}

// MakeBatches découpe texts en lots de batchSize lignes maximum, chaque lot
// portant jusqu'à contextSize lignes précédentes comme contexte.
func MakeBatches(texts []string, batchSize, contextSize int) []Batch {
	if batchSize < 1 {
		batchSize = 1
	}
	if contextSize < 0 {
		contextSize = 0
	}
	var batches []Batch
	for start := 0; start < len(texts); start += batchSize {
		end := start + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		ctxStart := start - contextSize
		if ctxStart < 0 {
			ctxStart = 0
		}
		batches = append(batches, Batch{
			Context: cloneSlice(texts[ctxStart:start]),
			Lines:   cloneSlice(texts[start:end]),
			Start:   start,
		})
	}
	return batches
}

func cloneSlice(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return append([]string(nil), s...)
}
