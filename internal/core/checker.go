package core

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gnomegl/usrsx/internal/client"
)

type Checker struct {
	client       *client.HTTPClient
	wmn          *WMNData
	maxTasks     int
	semaphore    chan struct{}
	progressChan chan SiteResult
}

func NewChecker(httpClient *client.HTTPClient, wmnData *WMNData, maxTasks int) *Checker {
	return &Checker{
		client:    httpClient,
		wmn:       wmnData,
		maxTasks:  maxTasks,
		semaphore: make(chan struct{}, maxTasks),
	}
}

func (ch *Checker) CheckSite(site Site, username string, fuzzyMode bool) SiteResult {
	result := SiteResult{
		SiteName:  site.Name,
		Category:  site.Category,
		Username:  username,
		CreatedAt: time.Now(),
	}

	if site.Name == "" {
		result.ResultStatus = ResultStatusError
		result.Error = "Site missing required field: name"
		return result
	}

	if site.Category == "" {
		result.ResultStatus = ResultStatusError
		result.Error = "Site missing required field: cat"
		return result
	}

	if site.URICheck == "" {
		result.ResultStatus = ResultStatusError
		result.Error = "Site missing required field: uri_check"
		return result
	}

	if !strings.Contains(site.URICheck, AccountPlaceholder) &&
		(site.PostBody == "" || !strings.Contains(site.PostBody, AccountPlaceholder)) {
		result.ResultStatus = ResultStatusError
		result.Error = fmt.Sprintf("Site '%s' missing %s placeholder", site.Name, AccountPlaceholder)
		return result
	}

	if fuzzyMode {
		if site.ECode == nil && site.EString == "" && site.MCode == nil && site.MString == "" {
			result.ResultStatus = ResultStatusError
			result.Error = "Site must define at least one matcher for fuzzy mode"
			return result
		}
	} else {
		if site.ECode == nil && site.EString == "" {
			result.ResultStatus = ResultStatusError
			result.Error = "Site missing required matchers for strict mode"
			return result
		}
		if site.MCode == nil && site.MString == "" {
			result.ResultStatus = ResultStatusError
			result.Error = "Site missing required matchers for strict mode"
			return result
		}
	}

	cleanUsername := username
	if site.StripBadChar != "" {
		for _, char := range site.StripBadChar {
			cleanUsername = strings.ReplaceAll(cleanUsername, string(char), "")
		}
	}

	if cleanUsername == "" {
		result.ResultStatus = ResultStatusError
		result.Error = fmt.Sprintf("Username '%s' became empty after character stripping", username)
		return result
	}

	uriCheck := strings.ReplaceAll(site.URICheck, AccountPlaceholder, cleanUsername)
	uriPretty := uriCheck
	if site.URIPretty != "" {
		uriPretty = strings.ReplaceAll(site.URIPretty, AccountPlaceholder, cleanUsername)
	}
	result.ResultURL = uriPretty

	start := time.Now()
	var resp *HTTPResponse
	var err error

	if site.PostBody != "" {
		postBody := strings.ReplaceAll(site.PostBody, AccountPlaceholder, cleanUsername)
		resp, err = ch.makeRequest(uriCheck, site.Headers, postBody)
	} else {
		resp, err = ch.makeRequest(uriCheck, site.Headers, "")
	}

	elapsed := time.Since(start).Seconds()
	result.Elapsed = elapsed

	if err != nil {
		result.ResultStatus = ResultStatusError
		result.Error = fmt.Sprintf("Network error: %v", err)
		return result
	}

	result.ResponseCode = resp.StatusCode
	result.ResponseText = resp.Body

	result.ResultStatus = GetResultStatus(
		resp.StatusCode,
		resp.Body,
		site.ECode,
		site.EString,
		site.MCode,
		site.MString,
		fuzzyMode,
	)

	if result.ResultStatus == ResultStatusFound {
		result.Metadata = ExtractMetadata(site.Name, resp.Body, resp.StatusCode)
	}

	return result
}

type HTTPResponse struct {
	StatusCode int
	Body       string
}

func (ch *Checker) makeRequest(url string, headers map[string]string, postBody string) (*HTTPResponse, error) {
	if postBody != "" {
		httpResp, httpErr := ch.client.Post(url, headers, postBody)
		if httpErr != nil {
			return nil, httpErr
		}
		body, readErr := client.ReadResponseBody(httpResp)
		if readErr != nil {
			return nil, readErr
		}
		return &HTTPResponse{
			StatusCode: httpResp.StatusCode,
			Body:       body,
		}, nil
	}

	httpResp, httpErr := ch.client.Get(url, headers)
	if httpErr != nil {
		return nil, httpErr
	}
	body, readErr := client.ReadResponseBody(httpResp)
	if readErr != nil {
		return nil, readErr
	}
	return &HTTPResponse{
		StatusCode: httpResp.StatusCode,
		Body:       body,
	}, nil
}

func (ch *Checker) CheckUsernames(usernames []string, sites []Site, fuzzyMode bool, progressChan chan<- SiteResult) []SiteResult {
	var wg sync.WaitGroup
	results := make([]SiteResult, 0)
	resultsMu := sync.Mutex{}

	for _, username := range usernames {
		for _, site := range sites {
			wg.Add(1)
			go func(u string, s Site) {
				defer wg.Done()

				ch.semaphore <- struct{}{}
				defer func() { <-ch.semaphore }()

				result := ch.CheckSite(s, u, fuzzyMode)

				if progressChan != nil {
					progressChan <- result
				}

				resultsMu.Lock()
				results = append(results, result)
				resultsMu.Unlock()
			}(username, site)
		}
	}

	wg.Wait()
	return results
}

func (ch *Checker) SelfCheck(sites []Site, fuzzyMode bool, progressChan chan<- SelfCheckResult) []SelfCheckResult {
	var wg sync.WaitGroup
	results := make([]SelfCheckResult, 0)
	resultsMu := sync.Mutex{}

	for _, site := range sites {
		if len(site.Known) == 0 {
			continue
		}

		wg.Add(1)
		go func(s Site) {
			defer wg.Done()

			selfCheckResult := SelfCheckResult{
				SiteName:  s.Name,
				Category:  s.Category,
				CreatedAt: time.Now(),
			}

			var siteResults []SiteResult
			for _, knownUser := range s.Known {
				ch.semaphore <- struct{}{}
				result := ch.CheckSite(s, knownUser, fuzzyMode)
				<-ch.semaphore

				siteResults = append(siteResults, result)
			}

			selfCheckResult.Results = siteResults
			selfCheckResult.OverallStatus = GetOverallStatus(siteResults, "")

			if progressChan != nil {
				progressChan <- selfCheckResult
			}

			resultsMu.Lock()
			results = append(results, selfCheckResult)
			resultsMu.Unlock()
		}(site)
	}

	wg.Wait()
	return results
}
