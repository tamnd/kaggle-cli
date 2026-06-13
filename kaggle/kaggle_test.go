package kaggle_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tamnd/kaggle-cli/kaggle"
)

func newTestClient(baseURL string) *kaggle.Client {
	cfg := kaggle.DefaultConfig()
	cfg.BaseURL = baseURL
	cfg.Rate = 0
	cfg.Retries = 3
	return kaggle.NewClient(cfg)
}

func TestGetSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	cfg := kaggle.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := kaggle.NewClient(cfg)

	start := time.Now()
	_, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestDatasetsParsesList(t *testing.T) {
	payload := []map[string]any{
		{
			"id":              1234,
			"ref":             "owner/dataset-name",
			"title":           "Dataset Name",
			"titleNullable":   "Dataset Name",
			"ownerName":       "Owner",
			"ownerNameNullable": "Owner",
			"ownerRef":        "owner",
			"ownerRefNullable": "owner",
			"licenseName":     "CC0",
			"licenseNameNullable": "CC0",
			"totalBytes":      int64(12345),
			"totalBytesNullable": int64(12345),
			"downloadCount":   999,
			"viewCount":       5000,
			"voteCount":       42,
			"kernelCount":     7,
			"usabilityRating": 0.9,
			"usabilityRatingNullable": 0.9,
			"currentVersionNumber": 2,
			"lastUpdated":     "2025-01-15T10:20:30.00Z",
			"url":             "https://www.kaggle.com/datasets/owner/dataset-name",
			"urlNullable":     "https://www.kaggle.com/datasets/owner/dataset-name",
			"tags":            []map[string]string{{"name": "ml", "ref": "ml"}, {"name": "nlp", "ref": "nlp"}},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	datasets, err := c.Datasets(context.Background(), kaggle.DatasetOptions{
		Search: "test",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("Datasets: %v", err)
	}
	if len(datasets) != 1 {
		t.Fatalf("got %d datasets, want 1", len(datasets))
	}
	d := datasets[0]
	if d.Rank != 1 {
		t.Errorf("rank = %d, want 1", d.Rank)
	}
	if d.ID != 1234 {
		t.Errorf("id = %d, want 1234", d.ID)
	}
	if d.Ref != "owner/dataset-name" {
		t.Errorf("ref = %q, want owner/dataset-name", d.Ref)
	}
	if d.Downloads != 999 {
		t.Errorf("downloads = %d, want 999", d.Downloads)
	}
	if d.Tags != "ml,nlp" {
		t.Errorf("tags = %q, want ml,nlp", d.Tags)
	}
	if d.LastUpdated != "2025-01-15T10:20:30Z" {
		t.Errorf("last_updated = %q, want 2025-01-15T10:20:30Z", d.LastUpdated)
	}
}

func TestCompetitionsParsesAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":401,"message":"Unauthenticated"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.Competitions(context.Background(), kaggle.CompetitionOptions{})
	if err == nil {
		t.Fatal("expected error for auth-required response, got nil")
	}
}

func TestCompetitionsParsesList(t *testing.T) {
	payload := []map[string]any{
		{
			"id":               42,
			"ref":              "titanic",
			"title":            "Titanic - Machine Learning from Disaster",
			"subtitle":         "Start here! Predict survival on the Titanic",
			"category":         "gettingStarted",
			"reward":           "Knowledge",
			"teamCount":        15000,
			"maxTeamSize":      3,
			"deadline":         "2030-01-01T00:00:00Z",
			"evaluationMetric": "Accuracy",
			"isKernelsOnly":    false,
			"url":              "https://www.kaggle.com/c/titanic",
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	comps, err := c.Competitions(context.Background(), kaggle.CompetitionOptions{})
	if err != nil {
		t.Fatalf("Competitions: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("got %d competitions, want 1", len(comps))
	}
	comp := comps[0]
	if comp.Rank != 1 {
		t.Errorf("rank = %d, want 1", comp.Rank)
	}
	if comp.Title != "Titanic - Machine Learning from Disaster" {
		t.Errorf("title = %q", comp.Title)
	}
	if comp.Teams != 15000 {
		t.Errorf("teams = %d, want 15000", comp.Teams)
	}
}
