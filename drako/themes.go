package main

// DracoThemeConfig holds the color palette for a theme.
type DracoThemeConfig struct {
	Primary    string
	Secondary  string
	Background string
	Foreground string
	Comment    string
	Cyan       string
	Green      string
	Orange     string
	Pink       string
	Purple     string
	Red        string
	Yellow     string
}

// themes is a map of available theme presets.
var themes = map[string]DracoThemeConfig{
	"dracula": {
		Primary:    "#ff2e63", // Hot Pink
		Secondary:  "#ff8c00", // Vibrant Orange
		Background: "#0d0221", // Deep Purple
		Foreground: "#f0f0f0", // Very light gray
		Comment:    "#5c527f", // Muted purple-gray
		Cyan:       "#00f5d4", // Bright aqua
		Green:      "#00f5d4", // Bright aqua (teal)
		Orange:     "#ff8c00", // Orange
		Pink:       "#ff2e63", // Hot Pink --selector?
		Purple:     "#9d4edd", // Vivid purple -- grid?
		Red:        "#ff2e63", // Hot pink-red
		Yellow:     "#f9f871", // Soft lemon
	},

	"dracula2":{               
		Primary:    "#ff79c6", // Hot pink
		Secondary:  "#bd93f9", // Soft purple
		Foreground: "#f8f8f2", // Off-white
		Background: "#282a36", // Deep slate
		Comment:    "#6272a4", // Muted blue-gray
		Cyan:       "#8be9fd", // Light cyan
		Green:      "#50fa7b", // Neon green
		Orange:     "#ffb86c", // Peach orange
		Pink:       "#ff79c6", // Hot pink
		Purple:     "#bd93f9", // Soft purple
		Red:        "#ff5555", // Bright red
		Yellow:     "#f1fa8c", // Pale yellow
		},

//"dracula": {
//        Primary:    "#ff2e63", // Hot Pink
//        Secondary:  "#ff8c00", // Vibrant Orange
//        Background: "#0d0221", // Deep Purple
//        Foreground: "#f0f0f0",
//        Comment:    "#5c527f",
//        Cyan:       "#00f5d4",
//        Green:      "#00f5d4",
//        Orange:     "#ff8c00",
//        Pink:       "#ff2e63",
//        Purple:     "#9d4edd",
//        Red:        "#ff2e63",
//        Yellow:     "#f9f871",
//},

	
	"jade": {
		Primary:    "#50fa7b", // Light Green (Logo Color)
		Secondary:  "#8be9fd", // Cyan
		Background: "#282a36",
		Foreground: "#f8f8f2",
		Comment:    "#6272a4",
		Cyan:       "#8be9fd", // Light cyan
		Green:      "#50fa7b",
		Orange:     "#ffb86c",
		Pink:       "#50fa7b",
		Purple:     "#50fa7b",
		Red:        "#ff5555",
		Yellow:     "#f1fa8c",
	},
	"nord": {
		Primary:    "#0077be", // Marine Blue (Logo Color)
		Secondary:  "#5e81ac", // Steel blue
		Background: "#0a192f", // Deep ocean navy
		Foreground: "#e5e9f0", // Very light gray
		Comment:    "#4c566a", // Gray-blue
		Cyan:       "#88c0d0", // Soft cyan
		Green:      "#a3be8c", // Sage green
		Orange:     "#d08770", // Muted orange
		Pink:       "#0077be", // Dusty mauve
		Purple:     "#0077be", // Dusty mauve
		Red:        "#bf616a", // Desaturated red
		Yellow:     "#ebcb8b", // Warm sand
	},
	"everforest": {
		Primary:    "#4a7c59", // Dark Green (Logo Color)
		Secondary:  "#a7c080", // Moss green
		Background: "#2d353b", // Charcoal
		Foreground: "#d3c6aa", // Pale khaki
		Comment:    "#5c6a72", // Grey
		Cyan:       "#83c092", // Desaturated teal
		Green:      "#a7c080", // Moss green
		Orange:     "#e69875", // Warm apricot
		Pink:       "#4a7c59", // Soft rose
		Purple:     "#4a7c59", // Soft rose
		Red:        "#e67e80", // Coral red
		Yellow:     "#dbbc7f", // Muted gold
	},
	"orasaka": {
		Primary:    "#f5c2e7", // Light Pink (Logo Color)
		Secondary:  "#cba6f7", // Mauve
		Background: "#1e1e2e", // Very dark slate
		Foreground: "#cdd6f4", // Light periwinkle
		Comment:    "#585b70", // Slate gray
		Cyan:       "#89dceb", // Sky cyan
		Green:      "#a6e3a1", // Mint green
		Orange:     "#fab387", // Apricot
		Pink:       "#f5c2e7",
		Purple:     "#cba6f7",
		Red:        "#f38ba8", // Rose red
		Yellow:     "#f9e2af", // Pale gold
	},
}

// UIColors describes concrete UI component colors derived from a theme.
type UIColors struct {
	HeaderFG        string
	FooterFG        string

	GridBorder      string
	GridSelBorder   string
	GridSelText     string

	Path            string
	PathSelected    string
	PathSeparator   string

	StatusInfo      string
	StatusPositive  string
	StatusNegative  string
	Warning         string

	HelpFG          string
	TitleFG         string
	ListHeaderFG    string
	CursorFG        string

	ButtonFG        string
	ButtonBG        string
	ButtonSelFG     string
	ButtonSelBG     string

	DropdownBorder  string
	DropdownFG      string
	DropdownBG      string
}

// mapThemeToUI maps a DracoThemeConfig to concrete UI component colors.
func mapThemeToUI(t DracoThemeConfig) UIColors {
	return UIColors{
		HeaderFG:       t.Primary,
		FooterFG:       t.Comment,

		GridBorder:     t.Comment,
		GridSelBorder:  t.Pink,
		GridSelText:    t.Pink,

		Path:           t.Primary,
		PathSelected:   t.Pink,
		PathSeparator:  t.Comment,

		StatusInfo:     t.Cyan,
		StatusPositive: t.Green,
		StatusNegative: t.Red,
		Warning:        t.Orange,

		HelpFG:         t.Comment,
		TitleFG:        t.Primary,
		ListHeaderFG:   t.Secondary,
		CursorFG:       t.Pink,

		ButtonFG:       t.Foreground,
		ButtonBG:       t.Comment,
		ButtonSelFG:    t.Background,
		ButtonSelBG:    t.Primary,

		DropdownBorder: t.Primary,
		DropdownFG:     t.Foreground,
		DropdownBG:     "#1a1a1a",
	}
}

// getTheme returns the color palette for a given theme name.
// If the theme is not found, it defaults to "dracula".
func getTheme(name string) DracoThemeConfig {
	if theme, ok := themes[name]; ok {
		return theme
	}
	return themes["dracula"]
}




















