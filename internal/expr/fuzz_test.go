package expr

import "testing"

func FuzzCompileAndEvaluate(f *testing.F) {
	for _, seed := range []string{"name", "value ?? 0", "items[0]", "true ? 'a' : 'b'", "({x: 1}).x", "[1, 2, 3].length"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		program, err := Compile(source)
		if err != nil {
			return
		}
		_, _ = program.Eval(&Context{Root: map[string]any{"name": "n", "value": 1, "items": []int{1}}})
	})
}
