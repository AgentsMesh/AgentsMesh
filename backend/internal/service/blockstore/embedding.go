package blockstoreservice

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
	"strings"
	"unicode"
)

type EmbeddingProvider interface {
	Model() string
	Dims() int
	Embed(ctx context.Context, text string) ([]float32, error)
}

type HashEmbedder struct {
	dims int
}

func NewHashEmbedder(dims int) *HashEmbedder {
	if dims <= 0 {
		dims = 256
	}
	return &HashEmbedder{dims: dims}
}

func (h *HashEmbedder) Model() string { return "hash-bow-v1" }
func (h *HashEmbedder) Dims() int     { return h.dims }

func (h *HashEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	vec := make([]float32, h.dims)
	for _, tok := range tokenize(text) {
		sum := sha256.Sum256([]byte(tok))
		idx := binary.BigEndian.Uint32(sum[:4]) % uint32(h.dims)
		sign := float32(1)
		if sum[4]&1 == 1 {
			sign = -1
		}
		vec[idx] += sign
	}
	return l2Normalize(vec), nil
}

// tokenize emits CJK ideographs per-codepoint so non-ASCII produces non-zero vectors —
// zero vectors poison pgvector cosine with NaN and break json.Marshal (issue #366).
func tokenize(s string) []string {
	s = strings.ToLower(s)
	capHint := len(s) / 3
	if capHint < 8 {
		capHint = 8
	}
	out := make([]string, 0, capHint)
	var buf strings.Builder
	flush := func() {
		if buf.Len() > 0 {
			out = append(out, buf.String())
			buf.Reset()
		}
	}
	for _, r := range s {
		if r < 0x80 {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				buf.WriteRune(r)
				continue
			}
			flush()
			continue
		}
		if isCJKIdeograph(r) {
			flush()
			out = append(out, string(r))
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			buf.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}

func isCJKIdeograph(r rune) bool {
	switch {
	case r >= 0x4E00 && r <= 0x9FFF:
		return true
	case r >= 0x3040 && r <= 0x309F:
		return true
	case r >= 0x30A0 && r <= 0x30FF:
		return true
	case r >= 0xAC00 && r <= 0xD7AF:
		return true
	}
	return false
}

func l2Normalize(v []float32) []float32 {
	var sq float64
	for _, x := range v {
		sq += float64(x) * float64(x)
	}
	if sq == 0 {
		return v
	}
	inv := float32(1.0 / math.Sqrt(sq))
	for i := range v {
		v[i] *= inv
	}
	return v
}

func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(na) * math.Sqrt(nb)))
}

func HashTextForEmbedding(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:8])
}
