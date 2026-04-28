package attribute

import "testing"

func TestFieldConstructorsAndConverters(t *testing.T) {
	t.Parallel()

	fields := []Field{
		Int("int", 1),
		IntSlice("ints", []int{1, 2}),
		Int64("int64", 3),
		Int64Slice("int64s", []int64{4, 5}),
		Float64("float", 1.5),
		Float64Slice("floats", []float64{2.5, 3.5}),
		String("string", "value"),
	}

	for _, field := range fields {
		if field.GetKey() == "" {
			t.Fatalf("expected key for %+v", field)
		}
		if ToZapField(field).Key != field.GetKey() {
			t.Fatalf("zap key mismatch for %s", field.GetKey())
		}
		if string(ToOtelAttribute(field).Key) != field.GetKey() {
			t.Fatalf("otel key mismatch for %s", field.GetKey())
		}
	}

	if got := ToOtelAttributes(fields); len(got) != len(fields) {
		t.Fatalf("expected %d otel attrs, got %d", len(fields), len(got))
	}
}
