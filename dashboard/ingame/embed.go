package ingame

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
)

//go:embed templates/*.html static/*
var files embed.FS

const AssetVersion = "v1.3.4"

type Renderer struct {
	templates *template.Template
}

func NewRenderer() (*Renderer, error) {
	functions := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"date": func(value int64) string {
			if value <= 0 {
				return ""
			}
			return time.Unix(value, 0).Format("2006-01-02")
		},
		"duration": formatDuration,
		"number":   formatInteger,
		"metricValue": func(metric service.IngameMetricDefinition, value float64) string {
			return service.FormatIngameValue(metric, value)
		},
		"markdown":   renderMarkdown,
		"badgeStyle": badgeStyle,
		"externalHref": func(value string) template.URL {
			const prefix = "steam://openurl_external/"
			if !strings.HasPrefix(value, prefix) || service.ValidateIngameURL(strings.TrimPrefix(value, prefix)) != nil {
				return ""
			}
			return template.URL(value)
		},
		"assetVersion": func() string {
			return AssetVersion
		},
	}
	templates, err := template.New("ingame").Funcs(functions).ParseFS(files, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse in-game templates: %w", err)
	}
	return &Renderer{templates: templates}, nil
}

func (r *Renderer) Render(writer io.Writer, name string, value any) error {
	return r.templates.ExecuteTemplate(writer, name, value)
}

func CSS() ([]byte, error) {
	return files.ReadFile("static/ingame.css")
}

func AchievementAtlas() ([]byte, error) {
	return files.ReadFile("static/achievements.png")
}

func badgeStyle(key string) template.CSS {
	position, ok := achievementAtlasPositions[key]
	if !ok {
		return "background-image:none"
	}
	return template.CSS(fmt.Sprintf("background-position:-%dpx -%dpx", position[0], position[1]))
}

func formatDuration(seconds int64) string {
	if seconds < 0 {
		seconds = 0
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	if hours > 0 {
		return fmt.Sprintf("%dh %02dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func formatInteger(value int64) string {
	negative := value < 0
	if negative {
		value = -value
	}
	raw := strconv.FormatInt(value, 10)
	var output strings.Builder
	if negative {
		output.WriteByte('-')
	}
	for index, character := range raw {
		if index > 0 && (len(raw)-index)%3 == 0 {
			output.WriteByte(',')
		}
		output.WriteRune(character)
	}
	return output.String()
}

var (
	rawHTMLPattern = regexp.MustCompile(`(?s)<[^>]*>`)
	imagePattern   = regexp.MustCompile(`!\[([^\]]*)\]\([^\)]*\)`)
	linkPattern    = regexp.MustCompile(`\[([^\]]+)\]\(([^\s\)]+)\)`)
	codePattern    = regexp.MustCompile("`([^`]+)`")
	strongPattern  = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	emPattern      = regexp.MustCompile(`\*([^*]+)\*`)
)

func renderMarkdown(markdown string) template.HTML {
	markdown = strings.ReplaceAll(markdown, "\r\n", "\n")
	markdown = strings.ReplaceAll(markdown, "\r", "\n")
	markdown = rawHTMLPattern.ReplaceAllString(markdown, "")
	markdown = imagePattern.ReplaceAllString(markdown, "$1")
	lines := strings.Split(markdown, "\n")
	var output bytes.Buffer
	inCode, inUL, inOL, inQuote := false, false, false, false
	closeLists := func() {
		if inUL {
			output.WriteString("</ul>")
			inUL = false
		}
		if inOL {
			output.WriteString("</ol>")
			inOL = false
		}
		if inQuote {
			output.WriteString("</blockquote>")
			inQuote = false
		}
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			closeLists()
			if inCode {
				output.WriteString("</code></pre>")
			} else {
				output.WriteString("<pre><code>")
			}
			inCode = !inCode
			continue
		}
		if inCode {
			template.HTMLEscape(&output, []byte(line+"\n"))
			continue
		}
		if trimmed == "" {
			closeLists()
			continue
		}
		if level, text, ok := markdownHeading(trimmed); ok {
			closeLists()
			fmt.Fprintf(&output, "<h%d>%s</h%d>", level, renderInline(text), level)
			continue
		}
		if strings.HasPrefix(trimmed, ">") {
			if inUL || inOL {
				closeLists()
			}
			if !inQuote {
				output.WriteString("<blockquote>")
				inQuote = true
			}
			output.WriteString("<p>" + renderInline(strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))) + "</p>")
			continue
		}
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			if inOL || inQuote {
				closeLists()
			}
			if !inUL {
				output.WriteString("<ul>")
				inUL = true
			}
			output.WriteString("<li>" + renderInline(strings.TrimSpace(trimmed[2:])) + "</li>")
			continue
		}
		if prefix := orderedListPrefix(trimmed); prefix > 0 {
			if inUL || inQuote {
				closeLists()
			}
			if !inOL {
				output.WriteString("<ol>")
				inOL = true
			}
			output.WriteString("<li>" + renderInline(strings.TrimSpace(trimmed[prefix:])) + "</li>")
			continue
		}
		closeLists()
		output.WriteString("<p>" + renderInline(trimmed) + "</p>")
	}
	closeLists()
	if inCode {
		output.WriteString("</code></pre>")
	}
	return template.HTML(output.String())
}

func renderInline(value string) string {
	var output strings.Builder
	last := 0
	for _, match := range linkPattern.FindAllStringSubmatchIndex(value, -1) {
		output.WriteString(renderInlineText(value[last:match[0]]))
		label := value[match[2]:match[3]]
		destination := value[match[4]:match[5]]
		if service.ValidateIngameURL(destination) == nil && destination != "" {
			output.WriteString(`<a href="` + template.HTMLEscapeString(destination) + `">` + renderInlineText(label) + `</a>`)
		} else {
			output.WriteString(renderInlineText(label))
		}
		last = match[1]
	}
	output.WriteString(renderInlineText(value[last:]))
	return output.String()
}

func renderInlineText(value string) string {
	escaped := template.HTMLEscapeString(value)
	escaped = codePattern.ReplaceAllString(escaped, "<code>$1</code>")
	escaped = strongPattern.ReplaceAllString(escaped, "<strong>$1</strong>")
	escaped = emPattern.ReplaceAllString(escaped, "<em>$1</em>")
	return escaped
}

func markdownHeading(value string) (int, string, bool) {
	level := 0
	for level < len(value) && level < 4 && value[level] == '#' {
		level++
	}
	if level == 0 || level >= len(value) || value[level] != ' ' {
		return 0, "", false
	}
	return level, strings.TrimSpace(value[level:]), true
}

func orderedListPrefix(value string) int {
	index := 0
	for index < len(value) && value[index] >= '0' && value[index] <= '9' {
		index++
	}
	if index == 0 || index+1 >= len(value) || value[index] != '.' || value[index+1] != ' ' {
		return 0
	}
	return index + 2
}

func markdownVisibleLength(value string) int {
	return utf8.RuneCountInString(rawHTMLPattern.ReplaceAllString(value, ""))
}
