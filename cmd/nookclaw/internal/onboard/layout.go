package onboard

import (
	"fmt"
	"os"
	"strings"
)

const layoutWidth = 62
const selectorVisibleOptions = 7
const selectorHeaderInnerWidth = 44

const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiBrand   = "\033[38;2;34;211;238m"
	ansiPrimary = "\033[38;2;209;213;219m"
	ansiDim     = "\033[38;2;107;114;128m"
	ansiSuccess = "\033[38;2;34;197;94m"
	ansiWarning = "\033[38;2;245;158;11m"
	ansiBorder  = "\033[38;2;55;65;81m"
)

func renderBanner(title string, subtitle ...string) string {
	var b strings.Builder
	b.WriteString(renderSelectorHeader())
	b.WriteString("\n\n")
	b.WriteString(styleTitle(title))
	b.WriteString("\n")
	for _, line := range subtitle {
		if strings.TrimSpace(line) == "" {
			continue
		}
		b.WriteString(styleSecondary(line))
		b.WriteString("\n")
	}
	return b.String()
}

func renderSummarySection(title string) string {
	rule := strings.Repeat("-", sectionRuleLength(title))
	return fmt.Sprintf("%s\n%s\n", styleTitle(title), styleBorder(rule))
}

func renderSectionHeader(step int, title string, intro string) string {
	label := fmt.Sprintf("%d. %s", step, title)

	var b strings.Builder
	b.WriteString(styleBrand(label))
	b.WriteString("\n")
	if strings.TrimSpace(intro) != "" {
		for _, line := range wrapText(intro, layoutWidth) {
			b.WriteString(styleSecondary(line))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	return b.String()
}

func renderCallout(title string, lines []string) string {
	var b strings.Builder
	b.WriteString(styleBrand(title))
	b.WriteString("\n")

	for _, raw := range lines {
		if strings.TrimSpace(raw) == "" {
			b.WriteString("  |\n")
			continue
		}

		firstIndent := "| "
		nextIndent := "| "
		if strings.HasPrefix(raw, "- ") {
			firstIndent = "| - "
			nextIndent = "|   "
			raw = strings.TrimPrefix(raw, "- ")
		}

		for _, line := range wrapIndentedText(raw, layoutWidth, firstIndent, nextIndent) {
			b.WriteString("  ")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	return b.String()
}

type selectorLine struct {
	Text string
	Role string
}

func renderSelectorScreen(context []selectorLine, actionLabel string, options []wizardOption, selected int) string {
	var b strings.Builder
	b.WriteString(renderSelectorHeader())
	b.WriteString("\n")

	for _, line := range context {
		if strings.TrimSpace(line.Text) == "" {
			b.WriteString("\n")
			continue
		}
		b.WriteString(styleForRole(line.Text, line.Role))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(stylePrimary(actionLabel))
	b.WriteString("\n\n")

	start, end := selectorWindow(len(options), selected, selectorVisibleOptions)
	if start > 0 {
		b.WriteString(styleSecondary(fmt.Sprintf("  ↑ %d more", start)))
		b.WriteString("\n")
	}

	for index := start; index < end; index++ {
		option := options[index]
		if index == selected {
			b.WriteString(styleBrand("❯ "))
		} else {
			b.WriteString("  ")
		}

		label := stylePrimary(option.Label)
		if option.Tone == "warning" {
			label = styleWarning(option.Label)
		}
		b.WriteString(label)
		b.WriteString("\n")

		if strings.TrimSpace(option.Description) != "" {
			for _, line := range wrapIndentedText(option.Description, layoutWidth, "  ", "  ") {
				b.WriteString(styleSecondary(line))
				b.WriteString("\n")
			}
		}
	}

	if end < len(options) {
		b.WriteString(styleSecondary(fmt.Sprintf("  ↓ %d more", len(options)-end)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleSecondary("Press ↑ ↓ to navigate • Enter to confirm"))
	b.WriteString("\n")
	return b.String()
}

func renderGuideScreen(title string, lines []string) string {
	var b strings.Builder
	b.WriteString(renderSelectorHeader())
	b.WriteString("\n\n")
	b.WriteString(styleTitle(title))
	b.WriteString("\n\n")

	for _, raw := range lines {
		if strings.TrimSpace(raw) == "" {
			b.WriteString("\n")
			continue
		}
		for _, line := range wrapText(raw, layoutWidth) {
			b.WriteString(styleSecondary(line))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	return b.String()
}

func renderSelectorHeader() string {
	rows := []struct {
		Text   string
		Styler func(string) string
	}{
		{Text: "", Styler: nil},
		{Text: "N O O K C L A W", Styler: styleHero},
		{Text: "private, local-first assistant", Styler: styleSecondary},
		{Text: "", Styler: nil},
	}

	var b strings.Builder
	border := strings.Repeat("─", selectorHeaderInnerWidth+2)
	b.WriteString(styleBorder("╭" + border + "╮"))
	b.WriteString("\n")
	for index, row := range rows {
		b.WriteString(renderHeaderRow(row.Text, row.Styler))
		if index < len(rows)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(styleBorder("╰" + border + "╯"))
	return b.String()
}

func renderHeaderRow(text string, styler func(string) string) string {
	content := centerText(text, selectorHeaderInnerWidth)
	if styler != nil && strings.TrimSpace(text) != "" {
		content = styler(content)
	}
	return styleBorder("│ ") + content + styleBorder(" │")
}

func centerText(text string, width int) string {
	runeCount := len([]rune(text))
	if runeCount >= width {
		return text
	}
	leftPadding := (width - runeCount) / 2
	rightPadding := width - runeCount - leftPadding
	return strings.Repeat(" ", leftPadding) + text + strings.Repeat(" ", rightPadding)
}

func wrapText(text string, width int) []string {
	return wrapIndentedText(text, width, "", "")
}

func wrapIndentedText(text string, width int, firstIndent string, nextIndent string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{firstIndent}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{firstIndent}
	}

	lines := []string{}
	currentIndent := firstIndent
	current := currentIndent

	for _, word := range words {
		if current == currentIndent {
			if len(currentIndent)+len(word) <= width {
				current += word
				continue
			}
			lines = append(lines, currentIndent+word)
			currentIndent = nextIndent
			current = currentIndent
			continue
		}

		if len(current)+1+len(word) <= width {
			current += " " + word
			continue
		}

		lines = append(lines, current)
		currentIndent = nextIndent
		current = currentIndent + word
	}

	if strings.TrimSpace(current) != "" {
		lines = append(lines, current)
	}

	return lines
}

func selectorWindow(total int, selected int, visible int) (int, int) {
	if total <= visible {
		return 0, total
	}

	start := selected - visible/2
	if start < 0 {
		start = 0
	}

	end := start + visible
	if end > total {
		end = total
		start = end - visible
	}

	return start, end
}

func styleBrand(value string) string {
	return applyStyle(value, ansiBrand)
}

func styleTitle(value string) string {
	return applyStyle(value, ansiBold+ansiPrimary)
}

func styleHero(value string) string {
	return applyStyle(value, ansiBold+ansiBrand)
}

func stylePrimary(value string) string {
	return applyStyle(value, ansiPrimary)
}

func styleSecondary(value string) string {
	return applyStyle(value, ansiDim)
}

func styleSuccess(value string) string {
	return applyStyle(value, ansiSuccess)
}

func styleWarning(value string) string {
	return applyStyle(value, ansiWarning)
}

func styleBorder(value string) string {
	return applyStyle(value, ansiBorder)
}

func styleForRole(value string, role string) string {
	switch role {
	case "success":
		return styleSuccess(value)
	case "secondary":
		return styleSecondary(value)
	case "warning":
		return styleWarning(value)
	case "brand":
		return styleBrand(value)
	case "raw":
		return value
	default:
		return stylePrimary(value)
	}
}

func applyStyle(value string, code string) string {
	if !supportsANSI() || strings.TrimSpace(value) == "" {
		return value
	}
	return code + value + ansiReset
}

func supportsANSI() bool {
	if strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("TERM")), "dumb") {
		return false
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func sectionRuleLength(title string) int {
	length := len(strings.TrimSpace(title))
	if length < 10 {
		return 10
	}
	if length > 18 {
		return 18
	}
	return length
}
