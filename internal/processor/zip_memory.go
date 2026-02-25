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

// UnzipInMemory extracts all XML files from a ZIP archive in memory.
// Returns a slice of raw XML byte slices (one per XML file in the archive).
func UnzipInMemory(zipData []byte) ([][]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el ZIP: %w", err)
	}

	var xmlFiles [][]byte
	for _, f := range r.File {
		if strings.ToLower(filepath.Ext(f.Name)) != ".xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		xmlFiles = append(xmlFiles, data)
	}
	return xmlFiles, nil
}

