package utils

import "testing"

func TestCapitalEach(t *testing.T) {
	tests := []struct {
		str  string
		want string
	}{
		{
			str:  "hello world",
			want: "Hello World",
		},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			if got := CapitalEach(tt.str); got != tt.want {
				t.Errorf("CapitalEach() = %v, want %v", got, tt.want)
			}
		})
	}
}
