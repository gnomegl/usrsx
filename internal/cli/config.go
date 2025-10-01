package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gnomegl/usrsx/internal/core"
)

type Config struct {
	Usernames     []string
	SiteNames     []string
	NoColor       bool
	NoProgressbar bool

	LocalLists   []string
	RemoteLists  []string
	LocalSchema  string
	RemoteSchema string

	SelfCheck bool

	IncludeCategories []string
	ExcludeCategories []string

	Proxy         string
	ProxyFile     string
	Timeout       int
	AllowRedirect bool
	VerifySSL     bool
	Impersonate   string

	MaxTasks    int
	FuzzyMode   bool
	ShowDetails bool
	Browse      bool

	SaveResponse bool
	ResponsePath string
	OpenResponse bool

	CSVExport  bool
	CSVPath    string
	PDFPath    string
	HTMLExport bool
	HTMLPath   string
	JSONExport bool
	JSONPath   string

	FilterAll       bool
	FilterErrors    bool
	FilterNotFound  bool
	FilterUnknown   bool
	FilterAmbiguous bool
}

func LoadWMNData(config *Config) (*core.WMNData, error) {
	var wmnData core.WMNData

	sources := append(config.RemoteLists, config.LocalLists...)
	if len(sources) == 0 {
		sources = []string{core.WMNRemoteURL}
	}

	allSites := make(map[string]core.Site)
	categoriesSet := make(map[string]bool)
	authorsSet := make(map[string]bool)
	var licenses []string

	for _, source := range sources {
		var data core.WMNData
		var err error

		if isURL(source) {
			data, err = loadFromURL(source)
		} else {
			data, err = loadFromFile(source)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to load WMN data from %s: %w", source, err)
		}

		for _, site := range data.Sites {
			allSites[site.Name] = site
		}
		for _, cat := range data.Categories {
			categoriesSet[cat] = true
		}
		for _, author := range data.Authors {
			authorsSet[author] = true
		}
		licenses = append(licenses, data.License...)
	}

	for _, site := range allSites {
		wmnData.Sites = append(wmnData.Sites, site)
	}
	for cat := range categoriesSet {
		wmnData.Categories = append(wmnData.Categories, cat)
	}
	for author := range authorsSet {
		wmnData.Authors = append(wmnData.Authors, author)
	}
	wmnData.License = licenses

	if len(wmnData.Sites) == 0 {
		return nil, fmt.Errorf("no sites loaded from any source")
	}

	if len(config.IncludeCategories) > 0 || len(config.ExcludeCategories) > 0 {
		wmnData.Sites = filterSitesByCategory(wmnData.Sites, config.IncludeCategories, config.ExcludeCategories)
	}

	return &wmnData, nil
}

func isURL(s string) bool {
	return len(s) > 7 && (s[:7] == "http://" || s[:8] == "https://")
}

func loadFromURL(url string) (core.WMNData, error) {
	resp, err := http.Get(url)
	if err != nil {
		return core.WMNData{}, err
	}
	defer resp.Body.Close()

	var data core.WMNData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return core.WMNData{}, err
	}

	return data, nil
}

func loadFromFile(path string) (core.WMNData, error) {
	file, err := os.Open(path)
	if err != nil {
		return core.WMNData{}, err
	}
	defer file.Close()

	var data core.WMNData
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return core.WMNData{}, err
	}

	return data, nil
}

func filterSitesByCategory(sites []core.Site, include, exclude []string) []core.Site {
	includeSet := make(map[string]bool)
	for _, cat := range include {
		includeSet[cat] = true
	}

	excludeSet := make(map[string]bool)
	for _, cat := range exclude {
		excludeSet[cat] = true
	}

	var filtered []core.Site
	for _, site := range sites {
		if len(include) > 0 && !includeSet[site.Category] {
			continue
		}
		if excludeSet[site.Category] {
			continue
		}
		filtered = append(filtered, site)
	}

	return filtered
}
