package kaggle

import "fmt"

// Dataset is the record emitted for a Kaggle dataset.
type Dataset struct {
	Rank        int     `json:"rank"`
	ID          int     `json:"id"`
	Ref         string  `json:"ref"`
	Title       string  `json:"title"`
	Subtitle    string  `json:"subtitle"`
	Owner       string  `json:"owner"`
	OwnerRef    string  `json:"owner_ref"`
	License     string  `json:"license"`
	SizeBytes   int64   `json:"size_bytes"`
	Downloads   int     `json:"downloads"`
	Views       int     `json:"views"`
	Votes       int     `json:"votes"`
	Notebooks   int     `json:"notebooks"`
	Usability   float64 `json:"usability"`
	Version     int     `json:"version"`
	LastUpdated string  `json:"last_updated"`
	Tags        string  `json:"tags"`
	URL         string  `json:"url"`
}

// Competition is the record emitted for a Kaggle competition.
type Competition struct {
	Rank         int    `json:"rank"`
	ID           int    `json:"id"`
	Ref          string `json:"ref"`
	Title        string `json:"title"`
	Subtitle     string `json:"subtitle"`
	Category     string `json:"category"`
	Reward       string `json:"reward"`
	Teams        int    `json:"teams"`
	MaxTeamSize  int    `json:"max_team_size"`
	Deadline     string `json:"deadline"`
	EvalMetric   string `json:"eval_metric"`
	IsKernelsOnly bool   `json:"kernels_only"`
	URL          string `json:"url"`
}

// ─── wire types from Kaggle API ───────────────────────────────────────────────

// apiDataset is the raw JSON shape from /api/v1/datasets/list.
type apiDataset struct {
	ID                       int          `json:"id"`
	Ref                      string       `json:"ref"`
	Title                    string       `json:"title"`
	TitleNullable            string       `json:"titleNullable"`
	Subtitle                 string       `json:"subtitle"`
	SubtitleNullable         string       `json:"subtitleNullable"`
	OwnerName                string       `json:"ownerName"`
	OwnerNameNullable        string       `json:"ownerNameNullable"`
	OwnerRef                 string       `json:"ownerRef"`
	OwnerRefNullable         string       `json:"ownerRefNullable"`
	LicenseName              string       `json:"licenseName"`
	LicenseNameNullable      string       `json:"licenseNameNullable"`
	TotalBytes               int64        `json:"totalBytes"`
	TotalBytesNullable       int64        `json:"totalBytesNullable"`
	DownloadCount            int          `json:"downloadCount"`
	ViewCount                int          `json:"viewCount"`
	VoteCount                int          `json:"voteCount"`
	KernelCount              int          `json:"kernelCount"`
	UsabilityRating          float64      `json:"usabilityRating"`
	UsabilityRatingNullable  float64      `json:"usabilityRatingNullable"`
	CurrentVersionNumber     int          `json:"currentVersionNumber"`
	LastUpdated              string       `json:"lastUpdated"`
	URL                      string       `json:"url"`
	URLNullable              string       `json:"urlNullable"`
	Tags                     []apiTag     `json:"tags"`
}

// apiTag is a tag entry in a dataset response.
type apiTag struct {
	Name string `json:"name"`
	Ref  string `json:"ref"`
}

// apiCompetition is the raw JSON shape from /api/v1/competitions/list.
type apiCompetition struct {
	ID               int    `json:"id"`
	Ref              string `json:"ref"`
	Title            string `json:"title"`
	Subtitle         string `json:"subtitle"`
	Category         string `json:"category"`
	Reward           string `json:"reward"`
	TeamCount        int    `json:"teamCount"`
	MaxTeamSize      int    `json:"maxTeamSize"`
	Deadline         string `json:"deadline"`
	EvaluationMetric string `json:"evaluationMetric"`
	IsKernelsOnly    bool   `json:"isKernelsOnly"`
	URL              string `json:"url"`
}

// ─── conversion helpers ───────────────────────────────────────────────────────

func apiDatasetToDataset(d apiDataset, rank int) Dataset {
	title := d.Title
	if title == "" {
		title = d.TitleNullable
	}
	subtitle := d.Subtitle
	if subtitle == "" {
		subtitle = d.SubtitleNullable
	}
	owner := d.OwnerName
	if owner == "" {
		owner = d.OwnerNameNullable
	}
	ownerRef := d.OwnerRef
	if ownerRef == "" {
		ownerRef = d.OwnerRefNullable
	}
	license := d.LicenseName
	if license == "" {
		license = d.LicenseNameNullable
	}
	size := d.TotalBytes
	if size == 0 {
		size = d.TotalBytesNullable
	}
	usability := d.UsabilityRating
	if usability == 0 {
		usability = d.UsabilityRatingNullable
	}
	u := d.URL
	if u == "" {
		u = d.URLNullable
	}
	if u == "" && d.Ref != "" {
		u = fmt.Sprintf("https://www.kaggle.com/datasets/%s", d.Ref)
	}

	tags := make([]string, 0, len(d.Tags))
	for _, t := range d.Tags {
		n := t.Name
		if n == "" {
			n = t.Ref
		}
		if n != "" {
			tags = append(tags, n)
		}
	}
	tagStr := joinStrings(tags, ",")

	return Dataset{
		Rank:        rank,
		ID:          d.ID,
		Ref:         d.Ref,
		Title:       trimSpace(title),
		Subtitle:    trimSpace(subtitle),
		Owner:       trimSpace(owner),
		OwnerRef:    ownerRef,
		License:     license,
		SizeBytes:   size,
		Downloads:   d.DownloadCount,
		Views:       d.ViewCount,
		Votes:       d.VoteCount,
		Notebooks:   d.KernelCount,
		Usability:   usability,
		Version:     d.CurrentVersionNumber,
		LastUpdated: trimDate(d.LastUpdated),
		Tags:        tagStr,
		URL:         u,
	}
}

func apiCompetitionToCompetition(c apiCompetition, rank int) Competition {
	u := c.URL
	if u == "" && c.Ref != "" {
		u = fmt.Sprintf("https://www.kaggle.com/c/%s", c.Ref)
	}
	return Competition{
		Rank:         rank,
		ID:           c.ID,
		Ref:          c.Ref,
		Title:        c.Title,
		Subtitle:     c.Subtitle,
		Category:     c.Category,
		Reward:       c.Reward,
		Teams:        c.TeamCount,
		MaxTeamSize:  c.MaxTeamSize,
		Deadline:     trimDate(c.Deadline),
		EvalMetric:   c.EvaluationMetric,
		IsKernelsOnly: c.IsKernelsOnly,
		URL:          u,
	}
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	out := ss[0]
	for _, s := range ss[1:] {
		out += sep + s
	}
	return out
}

func trimSpace(s string) string {
	result := ""
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !prevSpace && result != "" {
				result += " "
			}
			prevSpace = true
		} else {
			result += string(r)
			prevSpace = false
		}
	}
	// trim trailing space
	for len(result) > 0 && (result[len(result)-1] == ' ') {
		result = result[:len(result)-1]
	}
	return result
}

// trimDate removes the sub-second and timezone noise from Kaggle's timestamps.
// "2026-05-10T23:12:10.07Z" -> "2026-05-10T23:12:10Z"
func trimDate(s string) string {
	if len(s) < 10 {
		return s
	}
	// just keep the first 19 chars (YYYY-MM-DDTHH:MM:SS) + Z
	for i, c := range s {
		if c == '.' {
			return s[:i] + "Z"
		}
	}
	return s
}
