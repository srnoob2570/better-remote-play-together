package diagnostics

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/srnoob2570/better-rpt/internal/i18n"
)

const docsBaseURL = "https://github.com/srnoob2570/better-remote-play-together/blob/main/"

// RenderJSON returns the machine-readable report with a trailing newline.
func RenderJSON(report Report) ([]byte, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// ValidateJSON checks the versioned report contract without adding a schema
// dependency. The full schema lives in schema.json.
func ValidateJSON(data []byte) error {
	var raw struct {
		SchemaVersion int               `json:"schema_version"`
		GeneratedAt   string            `json:"generated_at"`
		Version       string            `json:"version"`
		Format        string            `json:"format"`
		Language      string            `json:"language"`
		Summary       Summary           `json:"summary"`
		Checks        []json.RawMessage `json:"checks"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("decode report: %w", err)
	}
	if raw.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema_version %d", raw.SchemaVersion)
	}
	if raw.GeneratedAt == "" || raw.Version == "" || len(raw.Checks) == 0 {
		return fmt.Errorf("report is missing required fields")
	}
	switch raw.Format {
	case "deb", "appimage", "dev":
	default:
		return fmt.Errorf("invalid report format %q", raw.Format)
	}
	switch raw.Language {
	case "es", "en":
	default:
		return fmt.Errorf("invalid report language %q", raw.Language)
	}
	switch raw.Summary.Verdict {
	case StatusOK, StatusWarn, StatusFail:
	default:
		return fmt.Errorf("invalid report verdict %q", raw.Summary.Verdict)
	}
	if raw.Summary.Counts.OK < 0 || raw.Summary.Counts.Warn < 0 ||
		raw.Summary.Counts.Fail < 0 || raw.Summary.Counts.Skip < 0 {
		return fmt.Errorf("negative summary count")
	}
	if raw.Summary.Total < 1 || raw.Summary.Total != raw.Summary.Counts.OK+
		raw.Summary.Counts.Warn+raw.Summary.Counts.Fail+raw.Summary.Counts.Skip {
		return fmt.Errorf("summary total does not match counts")
	}
	for index, item := range raw.Checks {
		var check struct {
			ID       string   `json:"id"`
			Status   Status   `json:"status"`
			Title    string   `json:"title"`
			Message  string   `json:"message"`
			Evidence []string `json:"evidence"`
		}
		if err := json.Unmarshal(item, &check); err != nil {
			return fmt.Errorf("check %d: %w", index, err)
		}
		if check.ID == "" || check.Title == "" || check.Message == "" {
			return fmt.Errorf("check %d is missing id/title/message", index)
		}
		switch check.Status {
		case StatusOK, StatusWarn, StatusFail, StatusSkip:
		default:
			return fmt.Errorf("check %q has invalid status %q", check.ID, check.Status)
		}
		if check.Evidence == nil {
			return fmt.Errorf("check %q has null evidence", check.ID)
		}
	}
	return nil
}

// RenderMarkdown builds the copy-paste report for an issue.
func RenderMarkdown(report Report) string {
	tr := i18n.New(report.Language)
	var out strings.Builder
	out.WriteString("# " + tr.T("diag.markdown.title") + "\n\n")
	out.WriteString(tr.T("diag.markdown.generated") + ": " + report.GeneratedAt + "\n\n")
	out.WriteString(tr.T("diag.markdown.version") + ": " + report.Version + "  \n")
	out.WriteString(tr.T("diag.markdown.format") + ": " + report.Format + "  \n")
	out.WriteString(tr.T("diag.markdown.language") + ": " + report.Language + "\n\n")
	out.WriteString("## " + tr.T("diag.markdown.summary") + "\n\n")
	out.WriteString(tr.T("diag.markdown.verdict") + ": **" + tr.T("diag.status."+string(report.Summary.Verdict)) + "**\n\n")
	fmt.Fprintf(&out, "%s: %d / %d / %d / %d / %d\n\n",
		tr.T("diag.markdown.counts"),
		report.Summary.Counts.OK,
		report.Summary.Counts.Warn,
		report.Summary.Counts.Fail,
		report.Summary.Counts.Skip,
		report.Summary.Total)
	out.WriteString("## " + tr.T("diag.markdown.checks") + "\n\n")
	for _, check := range report.Checks {
		out.WriteString("### " + check.Title + " (`" + check.ID + "`)\n\n")
		out.WriteString("- **" + tr.T("diag.markdown.status") + ":** " +
			tr.T("diag.status."+string(check.Status)) + "\n")
		out.WriteString("- **" + tr.T("diag.markdown.message") + ":** " + check.Message + "\n")
		if len(check.Evidence) > 0 {
			out.WriteString("- **" + tr.T("diag.markdown.evidence") + ":**\n")
			for _, line := range check.Evidence {
				out.WriteString("  - `" + strings.ReplaceAll(line, "`", "'") + "`\n")
			}
		}
		if check.NextStep != "" {
			out.WriteString("- **" + tr.T("diag.markdown.next") + ":** " + check.NextStep + "\n")
		}
		if len(check.Docs) > 0 {
			out.WriteString("- **" + tr.T("diag.markdown.docs") + ":** ")
			for index, doc := range check.Docs {
				if index > 0 {
					out.WriteString(", ")
				}
				out.WriteString("[" + doc + "](" + docsBaseURL + doc + ")")
			}
			out.WriteString("\n")
		}
		out.WriteString("\n")
	}
	return out.String()
}
