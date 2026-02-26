// processor/zip_memory.go
package processor

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// UnzipResult holds the files extracted from a DIAN invoice ZIP.
// A ZIP may contain the XML document plus a PDF visualisation.
type UnzipResult struct {
	XMLFiles [][]byte
	PDFFiles [][]byte
}

// UnzipInMemoryAll extracts both XML and PDF files from a ZIP archive in memory.
func UnzipInMemoryAll(zipData []byte) (UnzipResult, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return UnzipResult{}, fmt.Errorf("no se pudo leer el ZIP: %w", err)
	}

	var result UnzipResult
	for _, f := range r.File {
		ext := strings.ToLower(filepath.Ext(f.Name))
		if ext != ".xml" && ext != ".pdf" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil || len(data) == 0 {
			continue
		}
		switch ext {
		case ".xml":
			result.XMLFiles = append(result.XMLFiles, data)
		case ".pdf":
			result.PDFFiles = append(result.PDFFiles, data)
		}
	}
	return result, nil
}

// UnzipInMemory extracts all XML files from a ZIP archive in memory.
// Kept for backward compatibility; delegates to UnzipInMemoryAll.
func UnzipInMemory(zipData []byte) ([][]byte, error) {
	res, err := UnzipInMemoryAll(zipData)
	return res.XMLFiles, err
}
