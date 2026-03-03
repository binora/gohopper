package reader

import "fmt"

// ReaderWay represents an OSM way (ordered sequence of node references).
type ReaderWay struct {
	ReaderElement
	Nodes []int64
}

func NewReaderWay(id int64) *ReaderWay {
	return &ReaderWay{
		ReaderElement: NewReaderElement(id, TypeWay),
		Nodes:         make([]int64, 0, 5),
	}
}

func (w *ReaderWay) GetNodes() []int64 { return w.Nodes }

func (w *ReaderWay) String() string {
	return fmt.Sprintf("Way id:%d, nodes:%d, tags:%v", w.id, len(w.Nodes), w.properties)
}
