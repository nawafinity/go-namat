package namat

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type packagePart struct {
	Header zip.FileHeader
	Data   []byte
}

type docxPackage struct {
	Parts map[string]*packagePart
	Order []string
}

func readPackage(data []byte) (*docxPackage, error) {
	return readPackageWithLimits(data, 256<<20, 1<<30, 10_000)
}

func readPackageWithLimits(data []byte, maxPartBytes, maxUncompressedBytes int64, maxParts int) (*docxPackage, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("%w: open DOCX package: %w", ErrInvalidTemplate, err)
	}
	if len(reader.File) > maxParts {
		return nil, fmt.Errorf("%w: DOCX package contains %d parts; limit is %d", ErrSecurityLimit, len(reader.File), maxParts)
	}
	pkg := &docxPackage{Parts: make(map[string]*packagePart), Order: make([]string, 0, len(reader.File))}
	var total int64
	for _, file := range reader.File {
		name := strings.ReplaceAll(file.Name, "\\", "/")
		if name == "" || strings.HasPrefix(name, "/") || path.Clean(name) != name || strings.HasPrefix(name, "../") {
			return nil, fmt.Errorf("%w: invalid OOXML package part name %q", ErrInvalidTemplate, file.Name)
		}
		if _, exists := pkg.Parts[name]; exists {
			return nil, fmt.Errorf("%w: duplicate OOXML package part %q", ErrInvalidTemplate, name)
		}
		if file.UncompressedSize64 > uint64(maxPartBytes) {
			return nil, fmt.Errorf("%w: package part %s exceeds size limit", ErrSecurityLimit, name)
		}
		total += int64(file.UncompressedSize64)
		if total > maxUncompressedBytes {
			return nil, fmt.Errorf("%w: DOCX uncompressed content exceeds size limit", ErrSecurityLimit)
		}
		stream, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open package part %s: %w", file.Name, err)
		}
		content, readErr := io.ReadAll(io.LimitReader(stream, maxPartBytes+1))
		closeErr := stream.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read package part %s: %w", file.Name, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close package part %s: %w", file.Name, closeErr)
		}
		if int64(len(content)) > maxPartBytes {
			return nil, fmt.Errorf("%w: package part %s exceeds size limit", ErrSecurityLimit, name)
		}
		header := file.FileHeader
		header.Name = name
		pkg.Parts[name] = &packagePart{Header: header, Data: content}
		pkg.Order = append(pkg.Order, name)
	}
	if _, ok := pkg.Parts["[Content_Types].xml"]; !ok {
		return nil, fmt.Errorf("%w: not an OOXML package: [Content_Types].xml is missing", ErrInvalidTemplate)
	}
	if _, ok := pkg.Parts["word/document.xml"]; !ok {
		return nil, fmt.Errorf("%w: not a Word DOCX package: word/document.xml is missing", ErrInvalidTemplate)
	}
	return pkg, nil
}

func (p *docxPackage) clone() *docxPackage {
	result := &docxPackage{Parts: make(map[string]*packagePart, len(p.Parts)), Order: append([]string(nil), p.Order...)}
	for name, part := range p.Parts {
		header := part.Header
		result.Parts[name] = &packagePart{Header: header, Data: append([]byte(nil), part.Data...)}
	}
	return result
}

func (p *docxPackage) bytes() ([]byte, error) {
	return p.bytesWithCompression(flate.BestSpeed)
}

func (p *docxPackage) bytesWithCompression(level int) ([]byte, error) {
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	writer.RegisterCompressor(zip.Deflate, func(destination io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(destination, level)
	})
	seen := make(map[string]bool, len(p.Parts))
	writePart := func(name string, part *packagePart) error {
		header := part.Header
		header.Name = name
		header.Flags &^= 0x8
		header.CRC32 = 0
		header.CompressedSize = 0
		header.CompressedSize64 = 0
		header.UncompressedSize = 0
		header.UncompressedSize64 = 0
		header.Extra = nil
		stream, err := writer.CreateHeader(&header)
		if err != nil {
			return err
		}
		_, err = stream.Write(part.Data)
		return err
	}
	for _, name := range p.Order {
		part, ok := p.Parts[name]
		if !ok {
			continue
		}
		if err := writePart(name, part); err != nil {
			return nil, fmt.Errorf("write package part %s: %w", name, err)
		}
		seen[name] = true
	}
	var extra []string
	for name := range p.Parts {
		if !seen[name] {
			extra = append(extra, name)
		}
	}
	sort.Strings(extra)
	for _, name := range extra {
		if err := writePart(name, p.Parts[name]); err != nil {
			return nil, fmt.Errorf("write package part %s: %w", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close DOCX package: %w", err)
	}
	return out.Bytes(), nil
}

func isTemplateXMLPart(name string, data []byte, openDelimiter string) bool {
	clean := filepath.ToSlash(name)
	return strings.HasPrefix(clean, "word/") && strings.HasSuffix(strings.ToLower(clean), ".xml") && bytes.Contains(data, []byte(openDelimiter))
}
