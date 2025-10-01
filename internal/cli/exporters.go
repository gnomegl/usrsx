package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/gnomegl/usrsx/internal/core"
)

type Exporter struct {
	Results   []core.SiteResult
	Usernames []string
	Timestamp time.Time
}

func NewExporter(results []core.SiteResult, usernames []string) *Exporter {
	return &Exporter{
		Results:   results,
		Usernames: usernames,
		Timestamp: time.Now(),
	}
}

func (e *Exporter) ExportCSV(path string) error {
	var writer *csv.Writer
	var file *os.File
	var err error

	if path == "" {
		writer = csv.NewWriter(os.Stdout)
	} else {
		file, err = os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create CSV file: %w", err)
		}
		defer file.Close()
		writer = csv.NewWriter(file)
	}
	defer writer.Flush()

	header := []string{"Username", "Site", "Category", "Status", "URL", "Response Code", "Elapsed", "Error", "Timestamp"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, result := range e.Results {
		row := []string{
			result.Username,
			result.SiteName,
			result.Category,
			string(result.ResultStatus),
			result.ResultURL,
			fmt.Sprintf("%d", result.ResponseCode),
			fmt.Sprintf("%.2f", result.Elapsed),
			result.Error,
			result.CreatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	if path != "" {
		fmt.Printf("Exported to CSV: %s\n", path)
	}
	return nil
}

func (e *Exporter) ExportJSON(path string) error {
	var encoder *json.Encoder
	var file *os.File
	var err error

	data := map[string]interface{}{
		"usernames": e.Usernames,
		"timestamp": e.Timestamp.Format(time.RFC3339),
		"results":   e.Results,
		"summary": map[string]int{
			"total":     len(e.Results),
			"found":     e.countByStatus(core.ResultStatusFound),
			"not_found": e.countByStatus(core.ResultStatusNotFound),
			"errors":    e.countByStatus(core.ResultStatusError),
			"unknown":   e.countByStatus(core.ResultStatusUnknown),
			"ambiguous": e.countByStatus(core.ResultStatusAmbiguous),
		},
	}

	if path == "" {
		encoder = json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(data); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	} else {
		file, err = os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create JSON file: %w", err)
		}
		defer file.Close()
		encoder = json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(data); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
		fmt.Printf("Exported to JSON: %s\n", path)
	}

	return nil
}

func (e *Exporter) ExportHTML(path string) error {
	if path == "" {
		path = fmt.Sprintf("usrsx_results_%s.html", e.Timestamp.Format("20060102_150405"))
	}

	tmpl := `
<html>
<head>
    <meta charset="UTF-8">
    <title>usrsx Results - {{.Timestamp}}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        h1 { color: #333; }
        .summary {
            background: white;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .summary-item {
            display: inline-block;
            margin-right: 20px;
            padding: 10px;
        }
        table {
            width: 100%;
            background: white;
            border-collapse: collapse;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        th {
            background: #4CAF50;
            color: white;
            font-weight: 600;
        }
        tr:hover { background: #f5f5f5; }
        .status-found { color: #4CAF50; font-weight: bold; }
        .status-not-found { color: #999; }
        .status-error { color: #f44336; }
        .status-unknown { color: #ff9800; }
        .status-ambiguous { color: #ff9800; }
        a { color: #2196F3; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <h1>usrsx Results</h1>
    
    <div class="summary">
        <h2>Summary</h2>
        <div class="summary-item"><strong>Usernames:</strong> {{.UsernamesStr}}</div>
        <div class="summary-item"><strong>Timestamp:</strong> {{.Timestamp}}</div>
        <div class="summary-item"><strong>Total:</strong> {{.Total}}</div>
        <div class="summary-item"><strong>Found:</strong> <span class="status-found">{{.Found}}</span></div>
        <div class="summary-item"><strong>Not Found:</strong> {{.NotFound}}</div>
        <div class="summary-item"><strong>Errors:</strong> <span class="status-error">{{.Errors}}</span></div>
    </div>

    <table>
        <thead>
            <tr>
                <th>Username</th>
                <th>Site</th>
                <th>Category</th>
                <th>Status</th>
                <th>URL</th>
                <th>Response</th>
                <th>Time</th>
            </tr>
        </thead>
        <tbody>
            {{range .Results}}
            <tr>
                <td>{{.Username}}</td>
                <td>{{.SiteName}}</td>
                <td>{{.Category}}</td>
                <td class="status-{{.ResultStatus}}">{{.ResultStatus}}</td>
                <td>{{if .ResultURL}}<a href="{{.ResultURL}}" target="_blank">{{.ResultURL}}</a>{{end}}</td>
                <td>{{.ResponseCode}}</td>
                <td>{{printf "%.2f" .Elapsed}}s</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</body>
</html>`

	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create HTML file: %w", err)
	}
	defer file.Close()

	data := struct {
		Usernames    []string
		UsernamesStr string
		Timestamp    string
		Total        int
		Found        int
		NotFound     int
		Errors       int
		Results      []core.SiteResult
	}{
		Usernames:    e.Usernames,
		UsernamesStr: fmt.Sprintf("%v", e.Usernames),
		Timestamp:    e.Timestamp.Format(time.RFC3339),
		Total:        len(e.Results),
		Found:        e.countByStatus(core.ResultStatusFound),
		NotFound:     e.countByStatus(core.ResultStatusNotFound),
		Errors:       e.countByStatus(core.ResultStatusError),
		Results:      e.Results,
	}

	if err := t.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}

	fmt.Printf("Exported to HTML: %s\n", path)
	return nil
}

func (e *Exporter) ExportPDF(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	fmt.Fprintf(file, "usrsx Results\n")
	fmt.Fprintf(file, "================\n\n")
	fmt.Fprintf(file, "Usernames: %v\n", e.Usernames)
	fmt.Fprintf(file, "Timestamp: %s\n", e.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(file, "Total Results: %d\n\n", len(e.Results))

	fmt.Fprintf(file, "Summary:\n")
	fmt.Fprintf(file, "  Found: %d\n", e.countByStatus(core.ResultStatusFound))
	fmt.Fprintf(file, "  Not Found: %d\n", e.countByStatus(core.ResultStatusNotFound))
	fmt.Fprintf(file, "  Errors: %d\n", e.countByStatus(core.ResultStatusError))
	fmt.Fprintf(file, "  Unknown: %d\n", e.countByStatus(core.ResultStatusUnknown))
	fmt.Fprintf(file, "  Ambiguous: %d\n\n", e.countByStatus(core.ResultStatusAmbiguous))

	fmt.Fprintf(file, "Detailed Results:\n")
	fmt.Fprintf(file, "=================\n\n")

	for _, result := range e.Results {
		fmt.Fprintf(file, "Username: %s\n", result.Username)
		fmt.Fprintf(file, "Site: %s (%s)\n", result.SiteName, result.Category)
		fmt.Fprintf(file, "Status: %s\n", result.ResultStatus)
		if result.ResultURL != "" {
			fmt.Fprintf(file, "URL: %s\n", result.ResultURL)
		}
		if result.ResponseCode > 0 {
			fmt.Fprintf(file, "Response Code: %d\n", result.ResponseCode)
		}
		if result.Elapsed > 0 {
			fmt.Fprintf(file, "Elapsed: %.2fs\n", result.Elapsed)
		}
		if result.Error != "" {
			fmt.Fprintf(file, "Error: %s\n", result.Error)
		}
		fmt.Fprintf(file, "\n")
	}

	absPath, _ := filepath.Abs(path)
	fmt.Printf("Exported to text file (PDF placeholder): %s\n", absPath)
	return nil
}

func (e *Exporter) countByStatus(status core.ResultStatus) int {
	count := 0
	for _, result := range e.Results {
		if result.ResultStatus == status {
			count++
		}
	}
	return count
}
