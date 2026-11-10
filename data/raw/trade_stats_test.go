package raw

import (
	"reflect"
	"testing"
)

// The response is read as TradeQueryGenerator writes it: entries sorted by text then id, line breaks as spaces, and
// an option stat's choices kept with their ids as the site serves them.
func TestDecodeTradeStats(t *testing.T) {
	got, err := decodeTradeStats([]byte(`{"result":[{"id":"explicit","label":"Explicit","entries":[
		{"id":"explicit.stat_2","text":"b\nc","type":"explicit"},
		{"id":"explicit.stat_3","text":"a","type":"explicit"},
		{"id":"explicit.stat_1","text":"a","type":"explicit"},
		{"id":"explicit.stat_5","text":"o #","type":"explicit","option":{"options":[{"id":1,"text":"x\ny"},{"id":"z","text":"z"}]}},
		{"id":"explicit.stat_4","text":"b\r\nd\re","type":"explicit"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []TradeStatCategory{{ID: "explicit", Label: "Explicit", Entries: []TradeStatEntry{
		{ID: "explicit.stat_1", Text: "a", Type: "explicit"},
		{ID: "explicit.stat_3", Text: "a", Type: "explicit"},
		{ID: "explicit.stat_2", Text: "b c", Type: "explicit"},
		{ID: "explicit.stat_4", Text: "b d e", Type: "explicit"},
		{ID: "explicit.stat_5", Text: "o #", Type: "explicit", Option: &TradeStatOptions{Options: []TradeStatOption{{ID: 1.0, Text: "x y"}, {ID: "z", Text: "z"}}}},
	}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if _, err := decodeTradeStats([]byte(`{"error":{"code":2,"message":"Invalid query"}}`)); err == nil {
		t.Error("an API error decoded as a stats list")
	}
}
