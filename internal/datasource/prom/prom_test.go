package prom

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/common/model"

	"github.com/qf-studio/grot/internal/datasource"
)

// stubServer returns a Prometheus-shaped API server serving canned responses
// keyed by request path suffix (/query or /query_range).
func stubServer(t *testing.T, instant, rng string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/query_range"):
			_, _ = w.Write([]byte(rng))
		case strings.HasSuffix(r.URL.Path, "/query"):
			_, _ = w.Write([]byte(instant))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestQueryInstant(t *testing.T) {
	const body = `{"status":"success","data":{"resultType":"vector","result":[
		{"metric":{"__name__":"pilot_active_prs","repo":"a"},"value":[1700000000,"3"]},
		{"metric":{"__name__":"pilot_active_prs","repo":"b"},"value":[1700000000,"5"]},
		{"metric":{"__name__":"pilot_active_prs","repo":"c"},"value":[1700000000,"NaN"]}
	]}}`
	srv := stubServer(t, body, "")
	defer srv.Close()

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	series, err := c.QueryInstant(context.Background(), datasource.Instant{
		Expr: "pilot_active_prs", Legend: "{{repo}}",
	})
	if err != nil {
		t.Fatalf("QueryInstant: %v", err)
	}
	// NaN sample dropped → 2 series.
	if len(series) != 2 {
		t.Fatalf("series: got %d, want 2", len(series))
	}
	if series[0].Legend != "a" || series[1].Legend != "b" {
		t.Errorf("legends: %q, %q", series[0].Legend, series[1].Legend)
	}
	if v, ok := series[0].Last(); !ok || v != 3 {
		t.Errorf("value[0]: %v (%v)", v, ok)
	}
}

func TestQueryRange(t *testing.T) {
	const body = `{"status":"success","data":{"resultType":"matrix","result":[
		{"metric":{"model":"opus"},"values":[[1700000000,"1"],[1700000060,"2"],[1700000120,"NaN"]]}
	]}}`
	srv := stubServer(t, "", body)
	defer srv.Close()

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	series, err := c.QueryRange(context.Background(), datasource.Range{Expr: "x", Legend: "{{model}}"})
	if err != nil {
		t.Fatalf("QueryRange: %v", err)
	}
	if len(series) != 1 || series[0].Legend != "opus" {
		t.Fatalf("series: %+v", series)
	}
	// Range queries keep NaN points positionally (a gap at a real timestamp) so
	// charts preserve the time axis — renderers skip NaN dots.
	if len(series[0].Points) != 3 {
		t.Fatalf("points: got %d, want 3 (NaN kept)", len(series[0].Points))
	}
	if !math.IsNaN(series[0].Points[2].V) {
		t.Errorf("third point should be NaN, got %v", series[0].Points[2].V)
	}
	// Last() skips the trailing NaN and returns the newest finite value.
	if v, ok := series[0].Last(); !ok || v != 2 {
		t.Errorf("Last() = %v,%v; want 2,true", v, ok)
	}
}

func TestExpandLegend(t *testing.T) {
	m := model.Metric{"__name__": "http_requests", "method": "GET", "code": "200"}
	cases := []struct{ tmpl, want string }{
		{"{{method}} {{code}}", "GET 200"},
		{"{{ method }}", "GET"},                             // whitespace trimmed
		{"{{missing}}", ""},                                 // absent label → empty
		{"static", "static"},                                // no placeholders
		{"{{method}", "{{method}"},                          // unterminated → literal
		{"", "http_requests{code=\"200\", method=\"GET\"}"}, // empty → full label set
	}
	for _, c := range cases {
		if got := expandLegend(c.tmpl, m); got != c.want {
			t.Errorf("expandLegend(%q) = %q, want %q", c.tmpl, got, c.want)
		}
	}

	// Single-label metric → bare name.
	if got := expandLegend("", model.Metric{"__name__": "up"}); got != "up" {
		t.Errorf("default single: %q", got)
	}
}
