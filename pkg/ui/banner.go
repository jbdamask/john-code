package ui

import (
	"fmt"
	"os"
	"os/user"
    "path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (u *UI) DrawBanner(modelName string) {
	// Styles
	borderColor := lipgloss.Color("#D97757") // Orange/Reddish from screenshot
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(80)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D7D7D")). // Grayish
		MarginLeft(2)

	// Get User
	currentUser, _ := user.Current()
	username := "User"
	if currentUser != nil {
        // Split full name or use username
        if currentUser.Name != "" {
            names := strings.Fields(currentUser.Name)
            if len(names) > 0 {
                username = names[0]
            }
        } else {
		    username = currentUser.Username
        }
	}

	welcomeMsg := fmt.Sprintf("Welcome back %s!", username)
	welcomeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Align(lipgloss.Center).
        Width(30).
        MarginTop(1)

    // Logo is pre-generated in logo.go as const Logo
    logoStyle := lipgloss.NewStyle().
        Align(lipgloss.Center).
        Width(30).
        MarginTop(1).
        MarginBottom(1)

	// Info (Model, CWD)
	cwd, _ := os.Getwd()
    // Truncate CWD if too long
    if len(cwd) > 40 {
        cwd = "~/.../" + filepath.Base(cwd)
    }
    
	infoBlock := fmt.Sprintf("%s • Claude Max\n%s", modelName, cwd)
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D7D7D")).
		Align(lipgloss.Center).
        Width(30)

	// Left Column
	leftCol := lipgloss.JoinVertical(
		lipgloss.Center,
		welcomeStyle.Render(welcomeMsg),
		logoStyle.Render(Logo),
		infoStyle.Render(infoBlock),
	)

	// Right Column (Tips & Activity)
	tipsHeader := lipgloss.NewStyle().Foreground(borderColor).Render("Tips for getting started")
	tipsBody := "Run /init to create a CLAUDE.md file wi..."
    
    activityHeader := lipgloss.NewStyle().Foreground(borderColor).MarginTop(1).Render("Recent activity")
    activityBody := "No recent activity" // TODO: Pull from session history

	rightCol := lipgloss.JoinVertical(
		lipgloss.Left,
		tipsHeader,
		tipsBody,
        lipgloss.NewStyle().Foreground(lipgloss.Color("#333333")).Render("─────────────────────────────"),
        activityHeader,
        activityBody,
	)
    
    // Layout
    content := lipgloss.JoinHorizontal(
        lipgloss.Top,
        leftCol,
        lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(borderColor).Margin(0, 2).Padding(0, 2).Render(rightCol),
    )

	// Render
    banner := borderStyle.Render(content)
    
    // Overlay Title on Border (lipgloss doesn't support this natively easily, so we print above or use a hack?
    // Claude Code puts it IN the border line. Lipgloss BorderTopLabel works!)
    // Wait, BorderTopLabel was added recently? Let's check version. 
    // If not available, we'll just print it above.
    
    // Assuming recent lipgloss
    // borderStyle.BorderTop(true).BorderForeground(borderColor) ...
    
    // Let's manually print the title "John Code v0.0.1" offset?
    // Or just print it above.
    fmt.Println(titleStyle.Render("John Code v0.0.1"))
	fmt.Println(banner)
}
