package serializers

import "testing"

type sample struct {
	Name string
	Age  int
}

func TestJSONRoundTrip(t *testing.T) {
	serializer := JSON[sample]{}
	data, err := serializer.Marshal(sample{Name: "Ada", Age: 37})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	got, err := serializer.Unmarshal(data)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.Name != "Ada" || got.Age != 37 {
		t.Fatalf("unexpected value: %#v", got)
	}
}

func TestGobRoundTrip(t *testing.T) {
	serializer := Gob[sample]{}
	data, err := serializer.Marshal(sample{Name: "Grace", Age: 85})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	got, err := serializer.Unmarshal(data)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.Name != "Grace" || got.Age != 85 {
		t.Fatalf("unexpected value: %#v", got)
	}
}
