package expr

import (
	"strings"
	"testing"
)

func FuzzCompileAndEvaluate(f *testing.F) {
	for _, seed := range []string{
		"name", "value ?? 0", "items[0]", "({x: 1}).x",
		"items | len", "name | default('missing')", "({x: [1, {y: 2}]}).x[1].y",
		strings.Repeat("!", 512) + "true", strings.Repeat("(", 128) + "1" + strings.Repeat(")", 128),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		program, err := CompileWithOptions(source, CompileOptions{MaxBytes: 16 << 10, MaxTokens: 2_048, MaxDepth: 128})
		if err != nil {
			return
		}
		_, _ = program.Eval(&Context{Root: map[string]any{"name": "n", "value": 1, "items": []int{1}}})
	})
}
