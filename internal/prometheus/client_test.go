package prometheus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/skyhook-io/radar/internal/errorlog"
)

func TestProbe(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		body           string
		want           bool
		wantEmptyEntry bool // expect the empty-instance warning to be recorded
	}{
		{
			name:       "healthy prometheus with targets",
			statusCode: http.StatusOK,
			body:       `{"status":"success","data":{"resultType":"vector","result":[{"metric":{"job":"prometheus"},"value":[1700000000,"1"]}]}}`,
			want:       true,
		},
		{
			name:           "empty instance returns success with zero results",
			statusCode:     http.StatusOK,
			body:           `{"status":"success","data":{"resultType":"vector","result":[]}}`,
			want:           false,
			wantEmptyEntry: true,
		},
		{
			name:       "non-prometheus 200 response (html)",
			statusCode: http.StatusOK,
			body:       `<html><body>Login</body></html>`,
			want:       false,
		},
		{
			name:       "prometheus error body with 200",
			statusCode: http.StatusOK,
			body:       `{"status":"error","errorType":"bad_data","error":"invalid query"}`,
			want:       false,
		},
		{
			name:       "non-200 status",
			statusCode: http.StatusInternalServerError,
			body:       `oops`,
			want:       false,
		},
		{
			name:       "401 unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       ``,
			want:       false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errorlog.Reset()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			c := &Client{httpClient: &http.Client{Timeout: 5 * time.Second}}
			got := c.probe(context.Background(), srv.URL)
			if got != tc.want {
				t.Fatalf("probe() = %v, want %v", got, tc.want)
			}

			gotEmptyEntry := false
			for _, e := range errorlog.GetEntries() {
				if e.Source == "prometheus" && e.Level == "warning" {
					gotEmptyEntry = true
				}
			}
			if gotEmptyEntry != tc.wantEmptyEntry {
				t.Fatalf("empty-instance warning recorded = %v, want %v", gotEmptyEntry, tc.wantEmptyEntry)
			}
		})
	}
}
