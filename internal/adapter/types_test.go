package adapter

import (
	"testing"
)

func TestIsValidShape(t *testing.T) {
	for _, s := range AllShapes() {
		if !IsValidShape(string(s)) {
			t.Errorf("IsValidShape(%q) = false, want true", s)
		}
	}
	if IsValidShape("Histogram") {
		t.Errorf("IsValidShape(Histogram) = true, want false")
	}
}

func TestDataPayloadValidate(t *testing.T) {
	cases := []struct {
		name    string
		payload DataPayload
		ok      bool
	}{
		{"scalar ok", DataPayload{Shape: ShapeScalar, Scalar: &ScalarPayload{Value: 1}}, true},
		{"scalar empty", DataPayload{Shape: ShapeScalar}, false},
		{"timeseries ok", DataPayload{Shape: ShapeTimeSeries, TimeSeries: &TimeSeriesPayload{}}, true},
		{"timeseries empty", DataPayload{Shape: ShapeTimeSeries}, false},
		{"categorical ok", DataPayload{Shape: ShapeCategorical, Categorical: &CategoricalPayload{}}, true},
		{"entitylist ok", DataPayload{Shape: ShapeEntityList, EntityList: &EntityListPayload{}}, true},
		{"unknown shape", DataPayload{Shape: "Histogram"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.payload.Validate()
			if (err == nil) != c.ok {
				t.Errorf("Validate() err=%v, ok-expected=%v", err, c.ok)
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	r.Register("foo", func(cfg Config) (DataSource, error) { return nil, nil })
	if _, ok := r.Lookup("foo"); !ok {
		t.Fatal("expected foo registered")
	}
	if _, ok := r.Lookup("bar"); ok {
		t.Fatal("bar should not be registered")
	}
	got := r.Types()
	if len(got) != 1 || got[0] != "foo" {
		t.Errorf("Types() = %v", got)
	}

	defer func() {
		if recover() == nil {
			t.Error("expected panic on duplicate register")
		}
	}()
	r.Register("foo", func(cfg Config) (DataSource, error) { return nil, nil })
}
