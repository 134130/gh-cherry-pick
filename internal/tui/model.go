package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/134130/gh-cherry-pick/gitobj"
	"github.com/134130/gh-cherry-pick/internal/log"
)

// --- Phase ---

type phase int

const (
	phaseLoading     phase = iota // Loading PR list
	phaseSelectPR                 // Select a PR
	phaseInputBranch              // Input target branch
	phaseRunning                  // Cherry-pick in progress
	phaseDone                     // Success
	phaseError                    // Failure
)

// --- Function types (dependency injection to avoid circular imports) ---

type ListPRsFunc func(ctx context.Context, state string) ([]*gitobj.PullRequest, error)
type RunFunc func(ctx context.Context, prNumber int, onto string) error

// --- Messages ---

type fetchPRsMsg struct {
	prs []*gitobj.PullRequest
	err error
}

type stepStartMsg struct {
	title string
}

type stepDoneMsg struct {
	duration time.Duration
}

type stepErrorMsg struct {
	err      error
	duration time.Duration
}

type logLineMsg struct {
	icon string
	text string
}

type cherryPickDoneMsg struct {
	err error
}

// --- Styles ---

var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	boldStyle    = lipgloss.NewStyle().Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(1))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(2))
	failStyle    = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(1))
	prNumStyle   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(6))
	cursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	authorStyle  = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
)

// --- Log line (accumulated during a step) ---

type viewLogLine struct {
	icon string
	text string
}

// --- Model ---

type Model struct {
	phase phase

	// PR selection
	prs    []*gitobj.PullRequest
	cursor int

	// Branch input
	textInput textinput.Model

	// Progress
	spinner        spinner.Model
	currentStep    string
	currentStepLog []viewLogLine

	// Dependencies (injected)
	listPRs ListPRsFunc
	runFunc RunFunc
	ctx     context.Context
	program *tea.Program

	// Config
	prNumber int
	onto     string

	// Terminal size
	width, height int

	// Result
	err error
}

var _ tea.Model = (*Model)(nil)

// NewInteractiveModel creates a model for interactive mode (PR selection + branch input).
func NewInteractiveModel(ctx context.Context, listPRs ListPRsFunc, runFunc RunFunc) *Model {
	return newModel(ctx, phaseLoading, listPRs, runFunc, 0, "")
}

// NewDirectModel creates a model for direct mode (skip selection, show progress only).
func NewDirectModel(ctx context.Context, runFunc RunFunc, prNumber int, onto string) *Model {
	return newModel(ctx, phaseRunning, nil, runFunc, prNumber, onto)
}

func newModel(ctx context.Context, initialPhase phase, listPRs ListPRsFunc, runFunc RunFunc, prNumber int, onto string) *Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "e.g. release/1.0, main"
	ti.CharLimit = 100

	return &Model{
		phase:     initialPhase,
		spinner:   sp,
		textInput: ti,
		listPRs:   listPRs,
		runFunc:   runFunc,
		ctx:       ctx,
		prNumber:  prNumber,
		onto:      onto,
	}
}

// SetProgram sets the tea.Program reference. Must be called after tea.NewProgram() and before p.Run().
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// --- tea.Model interface ---

func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick}

	switch m.phase {
	case phaseLoading:
		cmds = append(cmds, m.fetchPRsCmd())
	case phaseRunning:
		cmds = append(cmds, m.startCherryPick())
	}

	return tea.Batch(cmds...)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case fetchPRsMsg:
		return m.handleFetchPRs(msg)

	case stepStartMsg:
		m.currentStep = msg.title
		m.currentStepLog = nil
		return m, nil

	case stepDoneMsg:
		return m.handleStepDone(msg)

	case stepErrorMsg:
		return m.handleStepError(msg)

	case logLineMsg:
		return m.handleLogLine(msg)

	case cherryPickDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.phase = phaseError
			return m, tea.Sequence(
				tea.Println(fmt.Sprintf("\n  %s  %s", failStyle.Render("✘"), errorStyle.Render(msg.err.Error()))),
				tea.Quit,
			)
		}
		m.phase = phaseDone
		return m, tea.Sequence(
			tea.Println(fmt.Sprintf("\n  %s", successStyle.Bold(true).Render("✨ Done!"))),
			tea.Quit,
		)
	}

	// Forward other messages to text input when in branch input phase
	if m.phase == phaseInputBranch {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) View() tea.View {
	switch m.phase {
	case phaseLoading:
		return tea.NewView(fmt.Sprintf("\n  %s Loading merged pull requests...\n", m.spinner.View()))

	case phaseSelectPR:
		return tea.NewView(m.viewSelectPR())

	case phaseInputBranch:
		return tea.NewView(m.viewInputBranch())

	case phaseRunning:
		return tea.NewView(m.viewRunning())

	case phaseDone, phaseError:
		return tea.NewView("")
	}

	return tea.NewView("")
}

// --- Key handling ---

func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.phase {
	case phaseSelectPR:
		switch key {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.prs)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.prs) > 0 {
				m.prNumber = m.prs[m.cursor].Number
				m.phase = phaseInputBranch
				return m, m.textInput.Focus()
			}
		case "q":
			return m, tea.Quit
		}

	case phaseInputBranch:
		switch key {
		case "enter":
			val := strings.TrimSpace(m.textInput.Value())
			if val != "" {
				m.onto = val
				m.phase = phaseRunning
				m.textInput.Blur()
				header := fmt.Sprintf("  🍒 %s", titleStyle.Render(fmt.Sprintf("cherry-pick PR #%d → %s", m.prNumber, m.onto)))
				return m, tea.Sequence(tea.Println(header+"\n"), m.startCherryPick())
			}
		case "esc":
			m.phase = phaseSelectPR
			m.textInput.Blur()
			return m, nil
		default:
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

	case phaseDone, phaseError:
		return m, tea.Quit
	}

	return m, nil
}

// --- Message handlers ---

func (m *Model) handleFetchPRs(msg fetchPRsMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.phase = phaseError
		return m, tea.Sequence(
			tea.Println(fmt.Sprintf("  %s  %s", failStyle.Render("✘"), errorStyle.Render("Failed to load pull requests: "+msg.err.Error()))),
			tea.Quit,
		)
	}
	if len(msg.prs) == 0 {
		m.phase = phaseError
		return m, tea.Sequence(
			tea.Println(fmt.Sprintf("  %s  No merged pull requests found.", dimStyle.Render("•"))),
			tea.Quit,
		)
	}
	m.prs = msg.prs
	m.phase = phaseSelectPR
	return m, nil
}

func (m *Model) handleStepDone(msg stepDoneMsg) (tea.Model, tea.Cmd) {
	// Format completed step with all accumulated log lines
	var b strings.Builder
	fmt.Fprintf(&b, "  %s  %s  %s",
		successStyle.Render("✔"),
		m.currentStep,
		dimStyle.Render(formatDuration(msg.duration)))
	for _, ll := range m.currentStepLog {
		fmt.Fprintf(&b, "\n     %s %s", ll.icon, ll.text)
	}
	m.currentStep = ""
	m.currentStepLog = nil
	return m, tea.Println(b.String())
}

func (m *Model) handleStepError(msg stepErrorMsg) (tea.Model, tea.Cmd) {
	var b strings.Builder
	fmt.Fprintf(&b, "  %s  %s  %s",
		failStyle.Render("✘"),
		m.currentStep,
		dimStyle.Render(formatDuration(msg.duration)))
	for _, ll := range m.currentStepLog {
		fmt.Fprintf(&b, "\n     %s %s", ll.icon, ll.text)
	}
	m.currentStep = ""
	m.currentStepLog = nil
	return m, tea.Println(b.String())
}

func (m *Model) handleLogLine(msg logLineMsg) (tea.Model, tea.Cmd) {
	if m.currentStep != "" {
		// Accumulate log lines under the current step (shown in View)
		m.currentStepLog = append(m.currentStepLog, viewLogLine(msg))
		return m, nil
	}
	// Between-step log: print directly
	text := strings.TrimRight(msg.text, "\n")
	return m, tea.Println(fmt.Sprintf("  %s %s", msg.icon, text))
}

// --- Commands ---

func (m *Model) fetchPRsCmd() tea.Cmd {
	return func() tea.Msg {
		prs, err := m.listPRs(m.ctx, "merged")
		return fetchPRsMsg{prs: prs, err: err}
	}
}

func (m *Model) startCherryPick() tea.Cmd {
	return func() tea.Msg {
		tuiStep := newTUIStepRunner(m.program)
		tuiLogger := newTUILogger(m.program)

		ctx := CtxWithStepRunner(m.ctx, tuiStep)
		ctx = log.CtxWithCustomLogger(ctx, tuiLogger)

		err := m.runFunc(ctx, m.prNumber, m.onto)
		return cherryPickDoneMsg{err: err}
	}
}

// --- View renderers ---

func (m *Model) viewSelectPR() string {
	var b strings.Builder

	fmt.Fprintf(&b, "\n  %s\n\n", titleStyle.Render("🍒 gh cherry-pick"))
	b.WriteString("  Select a merged pull request:\n\n")

	// Calculate visible range
	visible := max(m.height-8, 5)
	if visible > len(m.prs) {
		visible = len(m.prs)
	}
	start := 0
	if m.cursor >= visible {
		start = m.cursor - visible + 1
	}
	end := start + visible
	if end > len(m.prs) {
		end = len(m.prs)
	}

	maxTitleLen := max(m.width-30, 20)

	for i := start; i < end; i++ {
		pr := m.prs[i]
		title := pr.Title
		if len(title) > maxTitleLen {
			title = title[:maxTitleLen-3] + "..."
		}
		prNum := fmt.Sprintf("#%-5d", pr.Number)

		if i == m.cursor {
			fmt.Fprintf(&b, "  %s %s %s  %s\n",
				cursorStyle.Render("❯"),
				cursorStyle.Render(prNum),
				cursorStyle.Render(title),
				authorStyle.Render("@"+pr.Author.Login))
		} else {
			fmt.Fprintf(&b, "    %s %s  %s\n",
				prNumStyle.Render(prNum),
				title,
				authorStyle.Render("@"+pr.Author.Login))
		}
	}

	fmt.Fprintf(&b, "\n  %s\n", helpStyle.Render("↑/↓ navigate · enter select · q quit"))
	return b.String()
}

func (m *Model) viewInputBranch() string {
	var b strings.Builder

	fmt.Fprintf(&b, "\n  %s\n\n", titleStyle.Render("🍒 gh cherry-pick"))

	pr := m.prs[m.cursor]
	fmt.Fprintf(&b, "  Selected: %s %s\n\n",
		prNumStyle.Render(fmt.Sprintf("#%d", pr.Number)),
		pr.Title)
	fmt.Fprintf(&b, "  Target branch: %s\n", m.textInput.View())
	fmt.Fprintf(&b, "\n  %s\n", helpStyle.Render("enter confirm · esc back · ctrl+c quit"))
	return b.String()
}

func (m *Model) viewRunning() string {
	if m.currentStep == "" {
		return fmt.Sprintf("\n  %s Starting...\n", m.spinner.View())
	}

	var b strings.Builder
	fmt.Fprintf(&b, "  %s  %s\n", m.spinner.View(), boldStyle.Render(m.currentStep+"..."))
	for _, ll := range m.currentStepLog {
		lines := strings.Split(strings.TrimRight(ll.text, "\n"), "\n")
		fmt.Fprintf(&b, "     %s %s\n", ll.icon, lines[0])
		for _, line := range lines[1:] {
			fmt.Fprintf(&b, "       %s\n", line)
		}
	}
	return b.String()
}

// --- Helpers ---

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
