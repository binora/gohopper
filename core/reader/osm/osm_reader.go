package osm

import (
	"fmt"
	"os"
)

type OSMReader struct {
	InputFile string
}

func NewOSMReader(inputFile string) *OSMReader {
	return &OSMReader{InputFile: inputFile}
}

func (r *OSMReader) ValidateInput() error {
	if r.InputFile == "" {
		return fmt.Errorf("graphhopper.datareader.file is empty")
	}
	if _, err := os.Stat(r.InputFile); err != nil {
		return fmt.Errorf("cannot open datareader.file %q: %w", r.InputFile, err)
	}
	return nil
}
