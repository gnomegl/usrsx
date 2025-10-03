package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gnomegl/usrsx/internal/cli"
	"github.com/gnomegl/usrsx/internal/client"
	"github.com/gnomegl/usrsx/internal/core"
	"github.com/gnomegl/usrsx/internal/utils"
)

var (
	config  cli.Config
	rootCmd = &cobra.Command{
		Use:     "usrsx [username...]",
		Short:   "Username availability checker across hundreds of websites",
		Long:    `usrsx is a powerful username enumeration tool that checks username availability across hundreds of websites using the WhatsMyName dataset.`,
		Version: core.Version,
		RunE:    runCheck,
	}
)

func init() {
	f := rootCmd.Flags()

	f.StringSliceVarP(&config.SiteNames, "site", "s", []string{}, "Specific site name(s) to check")
	f.StringSliceVarP(&config.LocalLists, "local-list", "l", []string{}, "Path(s) to local JSON file(s)")
	f.StringSliceVarP(&config.RemoteLists, "remote-list", "r", []string{}, "URL(s) to fetch remote lists")
	f.StringVarP(&config.LocalSchema, "local-schema", "L", "", "Path to local schema file")
	f.StringVarP(&config.RemoteSchema, "remote-schema", "R", core.WMNSchemaURL, "URL to fetch schema")
	f.BoolVarP(&config.SelfCheck, "self-check", "S", false, "Run self-check mode")

	f.StringSliceVarP(&config.IncludeCategories, "include-categories", "I", []string{}, "Include only these categories")
	f.StringSliceVarP(&config.ExcludeCategories, "exclude-categories", "E", []string{}, "Exclude these categories")
	f.BoolVarP(&config.FilterAll, "filter-all", "a", false, "Show all results")
	f.BoolVarP(&config.FilterErrors, "filter-errors", "e", false, "Show only errors")
	f.BoolVarP(&config.FilterNotFound, "filter-not-found", "n", false, "Show only not found")
	f.BoolVarP(&config.FilterUnknown, "filter-unknown", "u", false, "Show only unknown")
	f.BoolVarP(&config.FilterAmbiguous, "filter-ambiguous", "g", false, "Show only ambiguous")

	f.BoolVarP(&config.CSVExport, "csv", "c", false, "Output as CSV to stdout")
	f.StringVarP(&config.CSVPath, "csv-output", "", "", "Export to CSV file (path required)")
	f.BoolVarP(&config.JSONExport, "json", "j", false, "Output as JSON to stdout")
	f.StringVarP(&config.JSONPath, "json-output", "", "", "Export to JSON file (path required)")
	f.BoolVarP(&config.HTMLExport, "html", "H", false, "Export to HTML")
	f.StringVarP(&config.HTMLPath, "html-path", "T", "", "Custom HTML path")
	f.StringVarP(&config.PDFPath, "pdf", "", "", "Export to PDF (path required)")

	f.StringVarP(&config.Proxy, "proxy", "p", "", "Proxy server (http://proxy:port, socks5://proxy:port)")
	f.StringVarP(&config.ProxyFile, "proxy-file", "F", "", "File containing proxies (one per line)")
	f.IntVarP(&config.Timeout, "timeout", "t", core.HTTPRequestTimeoutSeconds, "Request timeout in seconds")
	f.BoolVarP(&config.AllowRedirect, "allow-redirects", "A", core.HTTPAllowRedirects, "Follow HTTP redirects")
	f.BoolVarP(&config.VerifySSL, "verify-ssl", "V", core.HTTPSSLVerify, "Verify SSL certificates")
	f.StringVarP(&config.Impersonate, "impersonate", "i", "chrome", "Browser to impersonate (chrome, firefox, safari, edge)")
	f.IntVarP(&config.MaxTasks, "max-tasks", "m", core.MaxConcurrentTasks, "Maximum concurrent tasks")

	f.BoolVarP(&config.FuzzyMode, "fuzzy", "f", false, "Enable fuzzy validation mode")
	f.BoolVarP(&config.ShowDetails, "show-details", "d", false, "Show detailed output")
	f.BoolVarP(&config.NoColor, "no-color", "C", false, "Disable colored output")
	f.BoolVarP(&config.NoProgressbar, "no-progressbar", "P", false, "Disable progress bar")
	f.BoolVarP(&config.Browse, "browse", "b", false, "Open found profiles in browser")
	f.BoolVarP(&config.SaveResponse, "save-response", "w", false, "Save HTTP responses")
	f.StringVarP(&config.ResponsePath, "response-path", "W", "", "Custom path for responses")
	f.BoolVarP(&config.OpenResponse, "open-response", "o", false, "Open saved responses")
}

func runCheck(cmd *cobra.Command, args []string) error {
	if !config.SelfCheck {
		if len(args) == 0 {
			return fmt.Errorf("at least one username is required")
		}
		config.Usernames = args
	}

	if err := utils.ValidateNumericValues(config.MaxTasks, config.Timeout); err != nil {
		return err
	}

	if config.Proxy != "" {
		if err := utils.ValidateProxy(config.Proxy); err != nil {
			return err
		}
	}

	if len(config.Usernames) > 0 {
		var err error
		config.Usernames, err = utils.ValidateUsernames(config.Usernames)
		if err != nil {
			return err
		}
	}

	wmnData, err := cli.LoadWMNData(&config)
	if err != nil {
		return fmt.Errorf("failed to load WMN data: %w", err)
	}
	if !isStdoutExport() {
		fmt.Printf("Loaded %d sites\n", len(wmnData.Sites))
	}

	sites := wmnData.Sites
	if len(config.SiteNames) > 0 {
		sites, err = utils.FilterSites(config.SiteNames, sites)
		if err != nil {
			return err
		}
		if !isStdoutExport() {
			fmt.Printf("Filtered to %d sites\n", len(sites))
		}
	}

	clientConfig := client.ClientConfig{
		Timeout:       config.Timeout,
		VerifySSL:     config.VerifySSL,
		AllowRedirect: config.AllowRedirect,
		Impersonate:   client.BrowserImpersonation(config.Impersonate),
		Proxy:         config.Proxy,
		ProxyFile:     config.ProxyFile,
	}

	httpClient, err := client.NewHTTPClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	checker := core.NewChecker(httpClient, wmnData, config.MaxTasks)

	var results []core.SiteResult

	if config.SelfCheck {
		results = runSelfCheck(checker, sites)
	} else {
		results = runUsernameCheck(checker, sites)
	}

	if shouldExport() && !config.JSONExport {
		exportResults(results)
	}

	if config.JSONExport {
		cli.StreamJSONSummary(results, config.Usernames)
	}

	return nil
}

func runUsernameCheck(checker *core.Checker, sites []core.Site) []core.SiteResult {
	totalChecks := len(config.Usernames) * len(sites)

	if !isStdoutExport() {
		fmt.Printf("\nChecking %d username(s) across %d sites (%d total checks)\n\n",
			len(config.Usernames), len(sites), totalChecks)
	}

	progressChan := make(chan core.SiteResult, totalChecks)
	results := make([]core.SiteResult, 0, totalChecks)

	go func() {
		checker.CheckUsernames(config.Usernames, sites, config.FuzzyMode, progressChan)
		close(progressChan)
	}()

	for result := range progressChan {
		results = append(results, result)
		if !isStdoutExport() {
			displayResult(result)
		} else if config.JSONExport {
			if shouldStreamJSON(result) {
				cli.StreamJSON(result)
			}
		}
	}

	if !isStdoutExport() {
		displaySummary(results)
	}
	return results
}

func runSelfCheck(checker *core.Checker, sites []core.Site) []core.SiteResult {
	if !isStdoutExport() {
		fmt.Printf("\nRunning self-check on %d sites\n\n", len(sites))
	}

	progressChan := make(chan core.SelfCheckResult, len(sites))
	allResults := make([]core.SiteResult, 0)

	go func() {
		selfCheckResults := checker.SelfCheck(sites, config.FuzzyMode, progressChan)
		for _, scr := range selfCheckResults {
			progressChan <- scr
		}
		close(progressChan)
	}()

	for selfCheckResult := range progressChan {
		if !isStdoutExport() {
			fmt.Println(cli.FormatSelfCheckResult(selfCheckResult, config.ShowDetails))
		} else if config.JSONExport {
			for _, result := range selfCheckResult.Results {
				if shouldStreamJSON(result) {
					cli.StreamJSON(result)
				}
			}
		}
		allResults = append(allResults, selfCheckResult.Results...)
	}

	if !isStdoutExport() {
		displaySummary(allResults)
	}
	return allResults
}

func displayResult(result core.SiteResult) {
	if !shouldDisplayResult(result) {
		return
	}

	fmt.Println(cli.FormatResult(result, config.ShowDetails))
}

func shouldDisplayResult(result core.SiteResult) bool {
	if config.FilterAll {
		return true
	}
	if config.FilterErrors && result.ResultStatus == core.ResultStatusError {
		return true
	}
	if config.FilterNotFound && result.ResultStatus == core.ResultStatusNotFound {
		return true
	}
	if config.FilterUnknown && result.ResultStatus == core.ResultStatusUnknown {
		return true
	}
	if config.FilterAmbiguous && result.ResultStatus == core.ResultStatusAmbiguous {
		return true
	}
	if !config.FilterErrors && !config.FilterNotFound && !config.FilterUnknown && !config.FilterAmbiguous {
		return result.ResultStatus == core.ResultStatusFound
	}
	return false
}

func shouldStreamJSON(result core.SiteResult) bool {
	if config.FilterAll {
		return true
	}
	if config.FilterErrors && result.ResultStatus == core.ResultStatusError {
		return true
	}
	if config.FilterNotFound && result.ResultStatus == core.ResultStatusNotFound {
		return true
	}
	if config.FilterUnknown && result.ResultStatus == core.ResultStatusUnknown {
		return true
	}
	if config.FilterAmbiguous && result.ResultStatus == core.ResultStatusAmbiguous {
		return true
	}
	if !config.FilterErrors && !config.FilterNotFound && !config.FilterUnknown && !config.FilterAmbiguous {
		return result.ResultStatus == core.ResultStatusFound || result.ResultStatus == core.ResultStatusAmbiguous
	}
	return false
}

func displaySummary(results []core.SiteResult) {
	found := 0
	notFound := 0
	errors := 0
	unknown := 0
	ambiguous := 0

	for _, r := range results {
		switch r.ResultStatus {
		case core.ResultStatusFound:
			found++
		case core.ResultStatusNotFound:
			notFound++
		case core.ResultStatusError:
			errors++
		case core.ResultStatusUnknown:
			unknown++
		case core.ResultStatusAmbiguous:
			ambiguous++
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Summary")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Total: %d\n", len(results))
	fmt.Printf("Found: %d\n", found)
	fmt.Printf("Not Found: %d\n", notFound)
	fmt.Printf("Errors: %d\n", errors)
	fmt.Printf("Unknown: %d\n", unknown)
	fmt.Printf("Ambiguous: %d\n", ambiguous)
	fmt.Println(strings.Repeat("=", 50))
}

func isStdoutExport() bool {
	return config.JSONExport || config.CSVExport
}

func shouldExport() bool {
	return config.CSVExport || config.CSVPath != "" || config.JSONExport || config.JSONPath != "" || config.HTMLExport || config.PDFPath != ""
}

func exportResults(results []core.SiteResult) {
	exporter := cli.NewExporter(results, config.Usernames)

	if config.CSVExport {
		if err := exporter.ExportCSV(""); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting CSV: %v\n", err)
		}
	}

	if config.CSVPath != "" {
		if err := exporter.ExportCSV(config.CSVPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting CSV: %v\n", err)
		}
	}

	if config.JSONExport {
		if err := exporter.ExportJSON(""); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting JSON: %v\n", err)
		}
	}

	if config.JSONPath != "" {
		if err := exporter.ExportJSON(config.JSONPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting JSON: %v\n", err)
		}
	}

	if config.HTMLExport {
		if err := exporter.ExportHTML(config.HTMLPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting HTML: %v\n", err)
		}
	}

	if config.PDFPath != "" {
		if err := exporter.ExportPDF(config.PDFPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting PDF: %v\n", err)
		}
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
