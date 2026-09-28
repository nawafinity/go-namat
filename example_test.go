package namat_test

import (
	"context"
	"io"

	"github.com/nawafinity/go-namat"
)

func ExampleCompile() {
	var templateBytes []byte // Read a .docx file here.
	template, err := namat.Compile(templateBytes, namat.Options{})
	if err != nil {
		return
	}
	_, _ = template.Render(context.Background(), map[string]any{"name": "Namat"})
}

func ExampleCompileReader() {
	var templateReader io.Reader // Open a .docx file here.
	template, err := namat.CompileReader(templateReader, namat.Options{})
	if err != nil {
		return
	}
	var output io.Writer
	_ = template.RenderTo(context.Background(), output, map[string]any{"name": "Namat"})
}

func ExampleImage() {
	image := namat.Image{
		Data:      []byte("image bytes"),
		Extension: "png",
		Width:     8,
		Height:    4,
		Alt:       "Chart",
	}
	_ = image
}
