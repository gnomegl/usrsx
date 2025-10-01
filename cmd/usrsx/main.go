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

var config cli.Config

var rootCmd = &cobra.Command{
	Use:   "usrsx [username...]",
	Short: "Username availability checker across hundreds of websites",
	Long: `usrsx is a powerful username enumeration tool that checks username 
availability across hundreds of websites using the WhatsMyName dataset.

Features:
  - Browser impersonation for accurate detection
  - Concurrent checking with goroutines
  - Proxy support (single proxy or proxies.txt file rotation)
  - Multiple export formats (CSV, JSON, HTML, PDF)
  - Self-check mode for validation
  - Category filtering`,
	Version: core.Version,
	RunE:    runCheck,
}

func init() {
	rootCmd.Flags().StringSliceVarP(&config.SiteNames, "site", "s", []string{}, "Specific site name(s) to check")
	rootCmd.Flags().BoolVarP(&config.NoColor, "no-color", "C", false, "Disable colored output")
	rootCmd.Flags().BoolVarP(&config.NoProgressbar, "no-progressbar", "P", false, "Disable progress bar")
	rootCmd.Flags().StringSliceVarP(&config.LocalLists, "local-list", "l", []string{}, "Path(s) to local JSON file(s)")
	rootCmd.Flags().StringSliceVarP(&config.RemoteLists, "remote-list", "r", []string{}, "URL(s) to fetch remote lists")
	rootCmd.Flags().StringVarP(&config.LocalSchema, "local-schema", "L", "", "Path to local schema file")
	rootCmd.Flags().StringVarP(&config.RemoteSchema, "remote-schema", "R", core.WMNSchemaURL, "URL to fetch schema")
	rootCmd.Flags().BoolVarP(&config.SelfCheck, "self-check", "S", false, "Run self-check mode")
	rootCmd.Flags().StringSliceVarP(&config.IncludeCategories, "include-categories", "I", []string{}, "Include only these categories")
	rootCmd.Flags().StringSliceVarP(&config.ExcludeCategories, "exclude-categories", "E", []string{}, "Exclude these categories")
	rootCmd.Flags().StringVarP(&config.Proxy, "proxy", "p", "", "Proxy server (http://proxy:port, socks5://proxy:port)")
	rootCmd.Flags().StringVarP(&config.ProxyFile, "proxy-file", "F", "", "File containing proxies (one per line)")
	rootCmd.Flags().IntVarP(&config.Timeout, "timeout", "t", core.HTTPRequestTimeoutSeconds, "Request timeout in seconds")
	rootCmd.Flags().BoolVarP(&config.AllowRedirect, "allow-redirects", "A", core.HTTPAllowRedirects, "Follow HTTP redirects")
	rootCmd.Flags().BoolVarP(&config.VerifySSL, "verify-ssl", "V", core.HTTPSSLVerify, "Verify SSL certificates")
	rootCmd.Flags().StringVarP(&config.Impersonate, "impersonate", "i", "chrome", "Browser to impersonate (chrome, firefox, safari, edge)")
	rootCmd.Flags().IntVarP(&config.MaxTasks, "max-tasks", "m", core.MaxConcurrentTasks, "Maximum concurrent tasks")
	rootCmd.Flags().BoolVarP(&config.FuzzyMode, "fuzzy", "f", false, "Enable fuzzy validation mode")
	rootCmd.Flags().BoolVarP(&config.ShowDetails, "show-details", "d", false, "Show detailed output")
	rootCmd.Flags().BoolVarP(&config.Browse, "browse", "b", false, "Open found profiles in browser")
	rootCmd.Flags().BoolVarP(&config.SaveResponse, "save-response", "w", false, "Save HTTP responses")
	rootCmd.Flags().StringVarP(&config.ResponsePath, "response-path", "W", "", "Custom path for responses")
	rootCmd.Flags().BoolVarP(&config.OpenResponse, "open-response", "o", false, "Open saved responses")
	rootCmd.Flags().BoolVarP(&config.CSVExport, "csv", "c", false, "Output as CSV to stdout")
	rootCmd.Flags().StringVarP(&config.CSVPath, "csv-output", "", "", "Export to CSV file (path required)")
	rootCmd.Flags().StringVarP(&config.PDFPath, "pdf", "", "", "Export to PDF (path required)")
	rootCmd.Flags().BoolVarP(&config.HTMLExport, "html", "H", false, "Export to HTML")
	rootCmd.Flags().StringVarP(&config.HTMLPath, "html-path", "T", "", "Custom HTML path")
	rootCmd.Flags().BoolVarP(&config.JSONExport, "json", "j", false, "Output as JSON to stdout")
	rootCmd.Flags().StringVarP(&config.JSONPath, "json-output", "", "", "Export to JSON file (path required)")
	rootCmd.Flags().BoolVarP(&config.FilterAll, "filter-all", "a", false, "Show all results")
	rootCmd.Flags().BoolVarP(&config.FilterErrors, "filter-errors", "e", false, "Show only errors")
	rootCmd.Flags().BoolVarP(&config.FilterNotFound, "filter-not-found", "n", false, "Show only not found")
	rootCmd.Flags().BoolVarP(&config.FilterUnknown, "filter-unknown", "u", false, "Show only unknown")
	rootCmd.Flags().BoolVarP(&config.FilterAmbiguous, "filter-ambiguous", "g", false, "Show only ambiguous")
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
	fmt.Printf("Loaded %d sites\n", len(wmnData.Sites))

	sites := wmnData.Sites
	if len(config.SiteNames) > 0 {
		sites, err = utils.FilterSites(config.SiteNames, sites)
		if err != nil {
			return err
		}
		fmt.Printf("Filtered to %d sites\n", len(sites))
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

	if shouldExport() {
		exportResults(results)
	}

	return nil
}

func runUsernameCheck(checker *core.Checker, sites []core.Site) []core.SiteResult {
	totalChecks := len(config.Usernames) * len(sites)

	fmt.Printf("\nChecking %d username(s) across %d sites (%d total checks)\n\n",
		len(config.Usernames), len(sites), totalChecks)

	progressChan := make(chan core.SiteResult, totalChecks)
	results := make([]core.SiteResult, 0, totalChecks)

	go func() {
		checker.CheckUsernames(config.Usernames, sites, config.FuzzyMode, progressChan)
		close(progressChan)
	}()

	for result := range progressChan {
		results = append(results, result)
		displayResult(result)
	}

	displaySummary(results)
	return results
}

func runSelfCheck(checker *core.Checker, sites []core.Site) []core.SiteResult {
	fmt.Printf("\nRunning self-check on %d sites\n\n", len(sites))

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
		fmt.Println(cli.FormatSelfCheckResult(selfCheckResult, config.ShowDetails))
		allResults = append(allResults, selfCheckResult.Results...)
	}

	displaySummary(allResults)
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

func shouldExport() bool {
	return config.CSVExport || config.CSVPath != "" || config.JSONExport || config.JSONPath != "" || config.HTMLExport || config.PDFPath != ""
}

func exportResults(results []core.SiteResult) {
	exporter := cli.NewExporter(results, config.Usernames)

	if config.CSVExport {
		if err := exporter.ExportCSV(""); err != nil {
			fmt.Printf("Error exporting CSV: %v\n", err)
		}
	}

	if config.CSVPath != "" {
		if err := exporter.ExportCSV(config.CSVPath); err != nil {
			fmt.Printf("Error exporting CSV: %v\n", err)
		}
	}

	if config.JSONExport {
		if err := exporter.ExportJSON(""); err != nil {
			fmt.Printf("Error exporting JSON: %v\n", err)
		}
	}

	if config.JSONPath != "" {
		if err := exporter.ExportJSON(config.JSONPath); err != nil {
			fmt.Printf("Error exporting JSON: %v\n", err)
		}
	}

	if config.HTMLExport {
		if err := exporter.ExportHTML(config.HTMLPath); err != nil {
			fmt.Printf("Error exporting HTML: %v\n", err)
		}
	}

	if config.PDFPath != "" {
		if err := exporter.ExportPDF(config.PDFPath); err != nil {
			fmt.Printf("Error exporting PDF: %v\n", err)
		}
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
