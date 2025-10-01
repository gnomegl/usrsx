package utils

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gnomegl/usrsx/internal/core"
)

func ValidateNumericValues(maxTasks, timeout int) error {
	if maxTasks < core.MinTasks || maxTasks > core.MaxTasksLimit {
		return core.NewConfigurationError(
			fmt.Sprintf("Invalid max_tasks: %d must be between %d and %d", maxTasks, core.MinTasks, core.MaxTasksLimit),
			nil,
		)
	}

	if timeout < core.MinTimeout || timeout > core.MaxTimeout {
		return core.NewConfigurationError(
			fmt.Sprintf("Invalid timeout: %d must be between %d and %d seconds", timeout, core.MinTimeout, core.MaxTimeout),
			nil,
		)
	}

	if maxTasks > core.HighConcurrencyThreshold && timeout < core.HighConcurrencyMinTimeout {
		fmt.Printf("Warning: High concurrency (%d tasks) with low timeout (%ds) may cause failures\n", maxTasks, timeout)
	}
	if maxTasks > core.ExtremeConcurrencyThreshold {
		fmt.Printf("Warning: Extremely high concurrency (%d tasks) may overwhelm servers\n", maxTasks)
	}
	if timeout < core.LowTimeoutWarningThreshold {
		fmt.Printf("Warning: Very low timeout (%ds) may cause legitimate requests to fail\n", timeout)
	}

	return nil
}

func ValidateProxy(proxy string) error {
	if proxy == "" {
		return nil
	}

	if !strings.HasPrefix(proxy, "http://") &&
		!strings.HasPrefix(proxy, "https://") &&
		!strings.HasPrefix(proxy, "socks5://") {
		return core.NewConfigurationError(
			"Invalid proxy: must be http://, https://, or socks5:// URL",
			nil,
		)
	}

	_, err := url.Parse(proxy)
	if err != nil {
		return core.NewConfigurationError(
			fmt.Sprintf("Invalid proxy URL: %v", err),
			err,
		)
	}

	return nil
}

func ValidateUsernames(usernames []string) ([]string, error) {
	seen := make(map[string]bool)
	var unique []string

	for _, u := range usernames {
		name := strings.TrimSpace(u)
		if name != "" && !seen[name] {
			seen[name] = true
			unique = append(unique, name)
		}
	}

	if len(unique) == 0 {
		return nil, core.NewValidationError("No valid usernames provided", nil)
	}

	return unique, nil
}

func FilterSites(siteNames []string, sites []core.Site) ([]core.Site, error) {
	if len(siteNames) == 0 {
		return sites, nil
	}

	siteMap := make(map[string]bool)
	for _, name := range siteNames {
		siteMap[name] = true
	}

	availableMap := make(map[string]core.Site)
	for _, site := range sites {
		availableMap[site.Name] = site
	}

	var missing []string
	for name := range siteMap {
		if _, exists := availableMap[name]; !exists {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return nil, core.NewDataError(
			fmt.Sprintf("Unknown site names: %v", missing),
			nil,
		)
	}

	var filtered []core.Site
	for _, site := range sites {
		if siteMap[site.Name] {
			filtered = append(filtered, site)
		}
	}

	return filtered, nil
}
