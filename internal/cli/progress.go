package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gnomegl/usrsx/internal/core"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	subtleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	headerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

type ResultTracker struct {
	Total     int
	Found     int
	NotFound  int
	Errors    int
	Unknown   int
	Ambiguous int
	Processed int
}

type ProgressModel struct {
	spinner       spinner.Model
	progress      progress.Model
	tracker       *ResultTracker
	currentSite   string
	done          bool
	quitting      bool
	noProgressbar bool
	foundResults  []core.SiteResult
}

func NewProgressModel(total int, noProgressbar bool) ProgressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	p := progress.New(
		progress.WithSolidFill("240"),
		progress.WithoutPercentage(),
	)
	p.Width = 40

	return ProgressModel{
		spinner:       s,
		progress:      p,
		noProgressbar: noProgressbar,
		tracker: &ResultTracker{
			Total: total,
		},
		foundResults: make([]core.SiteResult, 0),
	}
}

type tickMsg struct{}

func (m ProgressModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{}
	}))
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case ResultMsg:
		m.tracker.Processed++
		switch msg.Result.ResultStatus {
		case core.ResultStatusFound:
			m.tracker.Found++
		case core.ResultStatusNotFound:
			m.tracker.NotFound++
		case core.ResultStatusError:
			m.tracker.Errors++
		case core.ResultStatusAmbiguous:
			m.tracker.Ambiguous++
		case core.ResultStatusUnknown:
			m.tracker.Unknown++
		}
		m.currentSite = msg.Result.SiteName

		if msg.Result.ResultStatus == core.ResultStatusFound {
			m.foundResults = append(m.foundResults, msg.Result)
		}

		return m, nil

	case DoneMsg:
		m.done = true
		return m, tea.Quit

	case tickMsg:
		if !m.done {
			return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
				return tickMsg{}
			})
		}
	}

	return m, nil
}

func (m ProgressModel) View() string {
	if m.quitting || m.done {
		return ""
	}

	if m.noProgressbar {
		return ""
	}

	var b strings.Builder

	b.WriteString("\033[K")

	if m.tracker.Total > 0 {
		percent := float64(m.tracker.Processed) / float64(m.tracker.Total)
		b.WriteString(m.progress.ViewAs(percent))
		b.WriteString(fmt.Sprintf("  %d/%d  ", m.tracker.Processed, m.tracker.Total))
	}

	b.WriteString(fmt.Sprintf("%s %d  ", successStyle.Render("✓"), m.tracker.Found))
	b.WriteString(fmt.Sprintf("%s %d", subtleStyle.Render("✗"), m.tracker.NotFound))

	if m.tracker.Errors > 0 {
		b.WriteString(fmt.Sprintf("  %s %d", errorStyle.Render("!"), m.tracker.Errors))
	}
	if m.tracker.Unknown > 0 {
		b.WriteString(fmt.Sprintf("  %s %d", warningStyle.Render("?"), m.tracker.Unknown))
	}
	if m.tracker.Ambiguous > 0 {
		b.WriteString(fmt.Sprintf("  %s %d", warningStyle.Render("~"), m.tracker.Ambiguous))
	}

	b.WriteString(fmt.Sprintf("  %s", subtleStyle.Render("(q: quit)")))

	return b.String()
}

type ResultMsg struct {
	Result core.SiteResult
}

type DoneMsg struct{}

func formatCompactResult(result core.SiteResult) string {
	var icon string
	var style lipgloss.Style

	switch result.ResultStatus {
	case core.ResultStatusFound:
		icon = "✓"
		style = successStyle
	case core.ResultStatusNotFound:
		icon = "✗"
		style = subtleStyle
	case core.ResultStatusError:
		icon = "!"
		style = errorStyle
	case core.ResultStatusAmbiguous:
		icon = "~"
		style = warningStyle
	case core.ResultStatusUnknown:
		icon = "?"
		style = warningStyle
	}

	line := fmt.Sprintf("%s %s",
		style.Render(icon),
		result.SiteName)

	if result.ResultURL != "" {
		line += fmt.Sprintf("  %s", subtleStyle.Render(result.ResultURL))
	}

	return line
}

func FormatResult(result core.SiteResult, showDetails bool) string {
	var b strings.Builder

	switch result.ResultStatus {
	case core.ResultStatusFound:
		b.WriteString(successStyle.Render("✓ FOUND"))
	case core.ResultStatusNotFound:
		b.WriteString(subtleStyle.Render("✗ NOT FOUND"))
	case core.ResultStatusError:
		b.WriteString(errorStyle.Render("! ERROR"))
	case core.ResultStatusAmbiguous:
		b.WriteString(warningStyle.Render("~ AMBIGUOUS"))
	case core.ResultStatusUnknown:
		b.WriteString(warningStyle.Render("? UNKNOWN"))
	}

	b.WriteString(fmt.Sprintf(" | %s | %s", result.Username, result.SiteName))

	if result.ResultURL != "" && result.ResultStatus == core.ResultStatusFound {
		b.WriteString(fmt.Sprintf(" | %s", infoStyle.Render(result.ResultURL)))
	}

	if showDetails {
		if result.ResponseCode > 0 {
			b.WriteString(fmt.Sprintf(" | HTTP %d", result.ResponseCode))
		}
		if result.Elapsed > 0 {
			b.WriteString(fmt.Sprintf(" | %.2fs", result.Elapsed))
		}
		if result.Error != "" {
			b.WriteString(fmt.Sprintf(" | %s", errorStyle.Render(result.Error)))
		}
	}

	if result.Metadata != nil && result.ResultStatus == core.ResultStatusFound {
		metadata := FormatMetadata(result.Metadata)
		if metadata != "" {
			b.WriteString("\n")
			b.WriteString(metadata)
		}
	}

	return b.String()
}

func FormatMetadata(metadata *core.ProfileMetadata) string {
	var lines []string
	boxStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	if metadata.DisplayName != "" {
		lines = append(lines, boxStyle.Render("  ├─ ")+labelStyle.Render("Name: ")+valueStyle.Render(metadata.DisplayName))
	}

	if metadata.Bio != "" {
		bioPreview := metadata.Bio
		if len(bioPreview) > 100 {
			bioPreview = bioPreview[:97] + "..."
		}
		lines = append(lines, boxStyle.Render("  ├─ ")+labelStyle.Render("Bio: ")+valueStyle.Render(bioPreview))
	}

	if metadata.AvatarURL != "" {
		lines = append(lines, boxStyle.Render("  ├─ ")+labelStyle.Render("Avatar: ")+valueStyle.Render(metadata.AvatarURL))
	}

	if metadata.Location != "" {
		lines = append(lines, boxStyle.Render("  ├─ ")+labelStyle.Render("Location: ")+valueStyle.Render(metadata.Location))
	}

	if metadata.Website != "" {
		lines = append(lines, boxStyle.Render("  ├─ ")+labelStyle.Render("Website: ")+valueStyle.Render(metadata.Website))
	}

	if metadata.JoinDate != "" {
		lines = append(lines, boxStyle.Render("  ├─ ")+labelStyle.Render("Joined: ")+valueStyle.Render(metadata.JoinDate))
	}

	if metadata.FollowerCount > 0 {
		lines = append(lines, boxStyle.Render("  ├─ ")+labelStyle.Render("Followers: ")+valueStyle.Render(fmt.Sprintf("%d", metadata.FollowerCount)))
	}

	if metadata.FollowingCount > 0 {
		lines = append(lines, boxStyle.Render("  ├─ ")+labelStyle.Render("Following: ")+valueStyle.Render(fmt.Sprintf("%d", metadata.FollowingCount)))
	}

	if metadata.IsVerified {
		lines = append(lines, boxStyle.Render("  ├─ ")+labelStyle.Render("Verified: ")+successStyle.Render("✓ Yes"))
	}

	if len(metadata.AdditionalLinks) > 0 {
		lines = append(lines, boxStyle.Render("  ├─ ")+labelStyle.Render("Links:"))
		for key, value := range metadata.AdditionalLinks {
			lines = append(lines, boxStyle.Render("  │  ├─ ")+labelStyle.Render(key+": ")+valueStyle.Render(value))
		}
	}

	if len(metadata.CustomFields) > 0 && len(metadata.CustomFields) <= 5 {
		for key, value := range metadata.CustomFields {
			if key != "username" && value != "" && len(value) < 50 {
				lines = append(lines, boxStyle.Render("  ├─ ")+labelStyle.Render(key+": ")+valueStyle.Render(value))
			}
		}
	}

	if len(lines) > 0 {
		lastIdx := len(lines) - 1
		lines[lastIdx] = strings.Replace(lines[lastIdx], "├─", "└─", 1)
		return strings.Join(lines, "\n")
	}
	return ""
}

func FormatSelfCheckResult(result core.SelfCheckResult, showDetails bool) string {
	var b strings.Builder

	switch result.OverallStatus {
	case core.ResultStatusFound:
		b.WriteString(successStyle.Render("✓ PASSED"))
	case core.ResultStatusError:
		b.WriteString(errorStyle.Render("✗ FAILED"))
	default:
		b.WriteString(warningStyle.Render("? PARTIAL"))
	}

	b.WriteString(fmt.Sprintf(" | %s", result.SiteName))

	foundCount := 0
	for _, r := range result.Results {
		if r.ResultStatus == core.ResultStatusFound {
			foundCount++
		}
	}

	b.WriteString(fmt.Sprintf(" | %d/%d known accounts found", foundCount, len(result.Results)))

	if showDetails && result.Error != "" {
		b.WriteString(fmt.Sprintf(" | %s", errorStyle.Render(result.Error)))
	}

	return b.String()
}
