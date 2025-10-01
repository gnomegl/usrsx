package core

import (
	"time"
)

type ResultStatus string

const (
	ResultStatusFound     ResultStatus = "found"
	ResultStatusNotFound  ResultStatus = "not_found"
	ResultStatusError     ResultStatus = "error"
	ResultStatusUnknown   ResultStatus = "unknown"
	ResultStatusAmbiguous ResultStatus = "ambiguous"
	ResultStatusNotValid  ResultStatus = "not_valid"
)

type ProfileMetadata struct {
	DisplayName     string            `json:"display_name,omitempty"`
	Bio             string            `json:"bio,omitempty"`
	AvatarURL       string            `json:"avatar_url,omitempty"`
	Location        string            `json:"location,omitempty"`
	Website         string            `json:"website,omitempty"`
	JoinDate        string            `json:"join_date,omitempty"`
	FollowerCount   int               `json:"follower_count,omitempty"`
	FollowingCount  int               `json:"following_count,omitempty"`
	IsVerified      bool              `json:"is_verified,omitempty"`
	AdditionalLinks map[string]string `json:"additional_links,omitempty"`
	CustomFields    map[string]string `json:"custom_fields,omitempty"`
}

type SiteResult struct {
	SiteName     string           `json:"site_name"`
	Category     string           `json:"category"`
	Username     string           `json:"username"`
	ResultStatus ResultStatus     `json:"result_status"`
	ResultURL    string           `json:"result_url,omitempty"`
	ResponseCode int              `json:"response_code,omitempty"`
	ResponseText string           `json:"response_text,omitempty"`
	Metadata     *ProfileMetadata `json:"metadata,omitempty"`
	Elapsed      float64          `json:"elapsed,omitempty"`
	Error        string           `json:"error,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
}

type Site struct {
	Name         string            `json:"name"`
	Category     string            `json:"cat"`
	URICheck     string            `json:"uri_check"`
	URIPretty    string            `json:"uri_pretty,omitempty"`
	ECode        *int              `json:"e_code,omitempty"`
	EString      string            `json:"e_string,omitempty"`
	MCode        *int              `json:"m_code,omitempty"`
	MString      string            `json:"m_string,omitempty"`
	Known        []string          `json:"known,omitempty"`
	PostBody     string            `json:"post_body,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	StripBadChar string            `json:"strip_bad_char,omitempty"`
}

type WMNData struct {
	Sites      []Site   `json:"sites"`
	Categories []string `json:"categories"`
	Authors    []string `json:"authors"`
	License    []string `json:"license"`
}

type SelfCheckResult struct {
	SiteName      string       `json:"site_name"`
	Category      string       `json:"category"`
	Results       []SiteResult `json:"results"`
	OverallStatus ResultStatus `json:"overall_status"`
	Error         string       `json:"error,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
}

func GetResultStatus(responseCode int, responseText string, eCode *int, eString string, mCode *int, mString string, fuzzyMode bool) ResultStatus {
	var conditionFound bool
	var conditionNotFound bool

	if fuzzyMode {
		if eCode != nil && responseCode == *eCode {
			conditionFound = true
		}
		if eString != "" && contains(responseText, eString) {
			conditionFound = true
		}
		if mCode != nil && responseCode == *mCode {
			conditionNotFound = true
		}
		if mString != "" && contains(responseText, mString) {
			conditionNotFound = true
		}
	} else {
		conditionFound = true
		if eCode != nil {
			conditionFound = conditionFound && (responseCode == *eCode)
		}
		if eString != "" {
			conditionFound = conditionFound && contains(responseText, eString)
		}
		if eCode == nil && eString == "" {
			conditionFound = false
		}

		conditionNotFound = true
		if mCode != nil {
			conditionNotFound = conditionNotFound && (responseCode == *mCode)
		}
		if mString != "" {
			conditionNotFound = conditionNotFound && contains(responseText, mString)
		}
		if mCode == nil && mString == "" {
			conditionNotFound = false
		}
	}

	if conditionFound && conditionNotFound {
		return ResultStatusAmbiguous
	} else if conditionFound {
		return ResultStatusFound
	} else if conditionNotFound {
		return ResultStatusNotFound
	}
	return ResultStatusUnknown
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) > 0 && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func GetOverallStatus(results []SiteResult, err string) ResultStatus {
	if err != "" {
		return ResultStatusError
	}
	if len(results) == 0 {
		return ResultStatusUnknown
	}

	statuses := make(map[ResultStatus]bool)
	for _, r := range results {
		statuses[r.ResultStatus] = true
	}

	if statuses[ResultStatusError] {
		return ResultStatusError
	}
	if len(statuses) > 1 {
		return ResultStatusUnknown
	}

	for status := range statuses {
		return status
	}
	return ResultStatusUnknown
}
