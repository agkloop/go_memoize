package hash

import "testing"

func TestFNVCompatibility(t *testing.T) {
	tests := []struct {
		name string
		got  uint64
		want uint64
	}{
		{name: "string", got: String(Offset64, "foo"), want: 15902901984413996407},
		{name: "uint", got: Uint(Offset64, 42), want: 12638128926439346813},
		{name: "bool false", got: Bool(Offset64, false), want: 12638153115695167455},
		{name: "bool true", got: Bool(Offset64, true), want: 12638152016183539244},
		{name: "comparable string", got: Comparable(Offset64, "foo"), want: 15902901984413996407},
		{name: "comparable int", got: Comparable(Offset64, 42), want: 12638128926439346813},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("hash = %d, want %d", tt.got, tt.want)
			}
		})
	}
}
