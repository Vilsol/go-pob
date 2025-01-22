package calculator

import (
	"github.com/Vilsol/go-pob/moddb"
)

type BCol struct {
	Label string
	Key   string
}

type Entry struct {
	Columns []BCol
	Rows    []map[string]string
	Lines   []string
}

type Breakdown struct {
	ModDB  *moddb.ModDB
	Output map[string]float64
	Actor  *Actor

	Data map[string]*Entry
}

func NewBreakdown(modDB *moddb.ModDB, output map[string]float64, actor *Actor) *Breakdown {
	return &Breakdown{
		ModDB:  modDB,
		Output: output,
		Actor:  actor,
		Data:   make(map[string]*Entry),
	}
}

func (b *Breakdown) AddCol(key string, cols ...BCol) {
	if _, ok := b.Data[key]; !ok {
		b.Data[key] = &Entry{}
	}

	b.Data[key].Columns = append(b.Data[key].Columns, cols...)
}

func (b *Breakdown) AddRow(key string, rows ...map[string]string) {
	if _, ok := b.Data[key]; !ok {
		b.Data[key] = &Entry{}
	}

	b.Data[key].Rows = append(b.Data[key].Rows, rows...)
}

func (b *Breakdown) AddLine(key string, s string) {
	if _, ok := b.Data[key]; !ok {
		b.Data[key] = &Entry{}
	}

	b.Data[key].Lines = append(b.Data[key].Lines, s)
}

func (b *Breakdown) GetData() map[string]*Entry {
	return b.Data
}
