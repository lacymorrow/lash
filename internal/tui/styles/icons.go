package styles

const (
	CheckIcon    string = "✓"
	ErrorIcon    string = "×"
	WarningIcon  string = "⚠"
	InfoIcon     string = "ⓘ"
	HintIcon     string = "∵"
	SpinnerIcon  string = "..."
	LoadingIcon  string = "⟳"
	DocumentIcon string = "🖼"
	ModelIcon    string = "◇"

	// Tool call icons
	ToolPending string = "●"
	ToolSuccess string = "✓"
	ToolError   string = "×"

	BorderThin  string = "│"
	BorderThick string = "▌"
)

var SelectionIgnoreIcons = []string{
	// CheckIcon,
	// ErrorIcon,
	// WarningIcon,
	// InfoIcon,
	// HintIcon,
	// SpinnerIcon,
	// LoadingIcon,
	// DocumentIcon,
	// ModelIcon,
	//
	// // Tool call icons
	// ToolPending,
	// ToolSuccess,
	// ToolError,

	BorderThin,
	BorderThick,
	"│",                // common border
	"┃",                // heavy border
	"┆",                // dashed border
	"┊",                // dotted border
	"╎",                // thin dashed border
	"╏",                // thick dashed border
	"╭", "╮", "╯", "╰", // rounded corners
	"┌", "┐", "┘", "└", // square corners
}
