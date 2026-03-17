package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type leagueTUI struct {
	app      *tview.Application
	ctx      context.Context
	virtTime time.Time
	mu       sync.Mutex
	running  bool

	// Focus cycling: slug → season → list
	focusables []tview.Primitive
	focusIdx   int

	// UI components
	pages       *tview.Pages
	timeView    *tview.TextView
	stateView   *tview.TextView
	outputView  *tview.TextView
	slugInput   *tview.InputField
	seasonInput *tview.InputField
	statusBar   *tview.TextView
	list        *tview.List
}

func tuiCommand(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("tui", flag.ExitOnError)
	slug := fs.String("league", "test-league", "Initial league slug")
	season := fs.Int("season", 1, "Initial season number")
	fs.Parse(args)

	// Redirect fd 2 (stderr) to a log file for the entire TUI session.
	//
	// tcell renders the screen by writing escape sequences to /dev/tty.
	// If ANY code (zerolog, standard log, viper, pgx, etc.) writes to
	// os.Stderr or fd 2 while tcell has control of the terminal, those raw
	// bytes interleave with tcell's escape sequences in the kernel TTY buffer
	// and corrupt the entire screen — including panels we never touched.
	//
	// Redirecting fd 2 at the OS level is the only reliable fix because it
	// catches every writer regardless of which logger or package they use.
	const debugLogPath = "/tmp/league-tester-debug.log"
	debugLog, dlErr := os.OpenFile(debugLogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if dlErr == nil {
		origStderrFd, dupErr := unix.Dup(2)
		if dupErr == nil {
			unix.Dup2(int(debugLog.Fd()), 2) //nolint:errcheck
			defer func() {
				unix.Dup2(origStderrFd, 2) //nolint:errcheck
				unix.Close(origStderrFd)
				debugLog.Close()
			}()
		}
		// Also redirect zerolog so its output goes to the file rather than
		// the now-redirected fd 2, giving us readable debug output.
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: debugLog, NoColor: true})
	}

	t := &leagueTUI{
		app:      tview.NewApplication(),
		ctx:      ctx,
		virtTime: time.Now().UTC().Truncate(time.Hour),
	}

	t.build(*slug, *season)

	// Set LEAGUE_NOW and initial time display before the app loop starts.
	os.Setenv("LEAGUE_NOW", t.virtTime.Format(time.RFC3339))
	t.timeView.SetText(t.formatTimeDisplay())

	go t.refreshState()

	return t.app.EnableMouse(true).SetRoot(t.pages, true).Run()
}

// ── Layout ────────────────────────────────────────────────────────────────────

func (t *leagueTUI) build(slug string, season int) {
	t.pages = tview.NewPages()

	// Virtual time panel — must set Box methods before type-specific ones
	// to avoid losing the *TextView type in a chain.
	t.timeView = tview.NewTextView()
	t.timeView.SetDynamicColors(true)
	t.timeView.SetTextAlign(tview.AlignCenter)
	t.timeView.SetBorder(true)
	t.timeView.SetTitle(" Virtual Time (LEAGUE_NOW) ")
	t.timeView.SetTitleColor(tcell.ColorYellow)

	// League config inputs
	t.slugInput = tview.NewInputField()
	t.slugInput.SetLabel("Slug:   ")
	t.slugInput.SetText(slug)
	t.slugInput.SetFieldWidth(18)

	t.seasonInput = tview.NewInputField()
	t.seasonInput.SetLabel("Season: ")
	t.seasonInput.SetText(strconv.Itoa(season))
	t.seasonInput.SetFieldWidth(4)
	t.seasonInput.SetAcceptanceFunc(tview.InputFieldInteger)

	configPanel := tview.NewFlex()
	configPanel.SetDirection(tview.FlexRow)
	configPanel.AddItem(t.slugInput, 1, 0, false)
	configPanel.AddItem(t.seasonInput, 1, 0, false)
	configPanel.SetBorder(true)
	configPanel.SetTitle(" League Config ")
	configPanel.SetTitleColor(tcell.ColorGreen)

	// Action list
	t.list = t.buildActionList()

	// Left panel: time → config → actions
	leftPanel := tview.NewFlex()
	leftPanel.SetDirection(tview.FlexRow)
	leftPanel.AddItem(t.timeView, 5, 0, false)
	leftPanel.AddItem(configPanel, 4, 0, false)
	leftPanel.AddItem(t.list, 0, 1, true)

	// Right top: league state (inspect output)
	t.stateView = tview.NewTextView()
	t.stateView.SetScrollable(true)
	t.stateView.SetWrap(false)
	t.stateView.SetBorder(true)
	t.stateView.SetTitle(" League State ")
	t.stateView.SetTitleColor(tcell.ColorWhite)

	// Right bottom: operation output log
	t.outputView = tview.NewTextView()
	t.outputView.SetDynamicColors(true)
	t.outputView.SetScrollable(true)
	t.outputView.SetWrap(true)
	t.outputView.SetBorder(true)
	t.outputView.SetTitle(" Output Log ")
	t.outputView.SetTitleColor(tcell.ColorWhite)
	t.outputView.SetChangedFunc(func() { t.outputView.ScrollToEnd() })

	rightPanel := tview.NewFlex()
	rightPanel.SetDirection(tview.FlexRow)
	rightPanel.AddItem(t.stateView, 0, 3, false)
	rightPanel.AddItem(t.outputView, 0, 2, false)

	// Status bar
	t.statusBar = tview.NewTextView()
	t.statusBar.SetDynamicColors(true)
	t.statusBar.SetText(t.statusReady())

	mainFlex := tview.NewFlex()
	mainFlex.AddItem(leftPanel, 34, 0, true)
	mainFlex.AddItem(rightPanel, 0, 1, false)

	root := tview.NewFlex()
	root.SetDirection(tview.FlexRow)
	root.AddItem(mainFlex, 0, 1, true)
	root.AddItem(t.statusBar, 1, 0, false)

	t.pages.AddPage("main", root, true, true)

	t.focusables = []tview.Primitive{t.slugInput, t.seasonInput, t.list}
	t.focusIdx = 2
	t.app.SetFocus(t.list)
	t.app.SetInputCapture(t.handleGlobalKeys)
}

func (t *leagueTUI) buildActionList() *tview.List {
	list := tview.NewList()
	list.ShowSecondaryText(false)
	list.SetHighlightFullLine(true)
	list.SetBorder(true)
	list.SetTitle(" Actions ")
	list.SetTitleColor(tcell.ColorTeal)

	sec := func(label string) {
		list.AddItem("  "+label, "", 0, nil)
	}
	add := func(label string, fn func()) {
		list.AddItem("    "+label, "", 0, fn)
	}

	sec("── Time Control ─────────────────")
	add("− 1 Week", func() { t.shiftTime(-7 * 24 * time.Hour) })
	add("− 1 Day", func() { t.shiftTime(-24 * time.Hour) })
	add("− 1 Hour", func() { t.shiftTime(-time.Hour) })
	add("+ 1 Hour", func() { t.shiftTime(time.Hour) })
	add("+ 1 Day", func() { t.shiftTime(24 * time.Hour) })
	add("+ 1 Week", func() { t.shiftTime(7 * 24 * time.Hour) })
	add("Jump to date...", func() { t.showDateModal() })

	sec("── Setup ────────────────────────")
	add("Create Test Users (32)", func() {
		t.runOp("Create Test Users", func() error {
			return createTestUsers(t.ctx, 32, 1, t.usersFilePath())
		})
	})
	add("Create League", func() {
		slug := t.slugInput.GetText()
		t.runOp("Create League", func() error {
			return createTestLeague(t.ctx, slug, slug, 8, "/tmp/lt-league.json")
		})
	})
	add("Register Users", func() {
		t.runOp("Register Users", func() error {
			return registerTestUsers(t.ctx, t.slugInput.GetText(), t.getSeason(), t.usersFilePath())
		})
	})

	sec("── Season Lifecycle ─────────────")
	add("Open Registration", func() {
		t.runOp("Open Registration", func() error {
			return openRegistration(t.ctx, t.slugInput.GetText(), t.getSeason())
		})
	})
	add("Prepare Divisions", func() {
		t.runOp("Prepare Divisions", func() error {
			return prepareDivisions(t.ctx, t.slugInput.GetText(), t.getSeason())
		})
	})
	add("Start Season", func() {
		t.runOp("Start Season", func() error {
			return startSeason(t.ctx, t.slugInput.GetText(), t.getSeason())
		})
	})
	add("Close Season", func() {
		t.runOp("Close Season", func() error {
			return closeSeason(t.ctx, t.slugInput.GetText())
		})
	})

	sec("── Simulate Games ───────────────")
	add("Simulate 1 Game", func() { t.simGames(false, 1) })
	add("Simulate 5 Games", func() { t.simGames(false, 5) })
	add("Simulate 10 Games", func() { t.simGames(false, 10) })
	add("Simulate All Games", func() { t.simGames(true, 0) })

	sec("── View ─────────────────────────")
	add("Refresh State", func() { go t.refreshState() })
	add("Full Inspect → Output", func() { t.runInspect() })

	return list
}

// ── Actions ───────────────────────────────────────────────────────────────────

func (t *leagueTUI) simGames(all bool, n int) {
	label := fmt.Sprintf("Simulate %d Game(s)", n)
	if all {
		label = "Simulate All Games"
	}
	t.runOp(label, func() error {
		return simulateGames(t.ctx, t.slugInput.GetText(), t.getSeason(), all, n, 0)
	})
}

func (t *leagueTUI) shiftTime(d time.Duration) {
	t.mu.Lock()
	t.virtTime = t.virtTime.Add(d)
	ts := t.virtTime.Format(time.RFC3339)
	display := t.formatTimeDisplay()
	t.mu.Unlock()

	os.Setenv("LEAGUE_NOW", ts)
	// Called from a tview list handler (main goroutine) — update directly.
	// Never use QueueUpdateDraw from within a tview handler: it blocks on
	// <-ch waiting for the main loop, which is already blocked here → deadlock.
	t.timeView.SetText(display)
}

// runOp runs fn in a background goroutine, routing all output (zerolog +
// stdout) to the output panel. Only one op runs at a time.
//
// runOp is always called from a tview event handler (main goroutine), so the
// synchronous setup code must update primitives directly — never via
// QueueUpdateDraw, which would deadlock waiting for the main loop to process
// the queued function while the main loop is blocked here.
func (t *leagueTUI) runOp(name string, fn func() error) {
	if t.running {
		// Direct update — main goroutine context.
		fmt.Fprintln(t.outputView, "[red]⚠  busy — wait for current operation to finish[-]")
		return
	}
	t.running = true
	// Direct updates — main goroutine context.
	t.statusBar.SetText(fmt.Sprintf(" [yellow]⏳ %s...[-]  |  Tab=cycle focus  q=quit  ESC=close modal", name))
	fmt.Fprintf(t.outputView, "\n[teal]══ %s ══[-]  [gray]@ %s[-]\n",
		name, t.virtTime.Format("2006-01-02 15:04 UTC"))

	go func() {
		// Buffer ALL zerolog output so we can write it to the UI in one
		// QueueUpdateDraw call at the end.  Multiple concurrent QueueUpdateDraw
		// calls each trigger a tcell redraw; doing many in rapid succession
		// corrupts the front/back buffer diff and garbles the screen.
		var logBuf bytes.Buffer
		origLogger := log.Logger
		log.Logger = zerolog.New(
			zerolog.ConsoleWriter{Out: &logBuf, NoColor: true, TimeFormat: "15:04:05"},
		).With().Timestamp().Logger()

		err := fn()

		log.Logger = origLogger

		t.app.QueueUpdateDraw(func() {
			if logBuf.Len() > 0 {
				fmt.Fprint(t.outputView, logBuf.String())
			}
			if err != nil {
				fmt.Fprintf(t.outputView, "[red]✗ ERROR: %v[-]\n", err)
			} else {
				fmt.Fprintln(t.outputView, "[green]✓ Done[-]")
			}
		})

		t.running = false
		t.setStatus(t.statusReady())
		go t.refreshState()
	}()
}

// captureInspect runs inspectLeague with stdout captured to a string.
// Must be called from a background goroutine — it blocks until inspect is done.
func (t *leagueTUI) captureInspect(slug string) (string, error) {
	pr, pw, err := os.Pipe()
	if err != nil {
		return "", err
	}

	origStdout := os.Stdout
	os.Stdout = pw

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		io.Copy(&buf, pr) //nolint:errcheck
		close(done)
	}()

	origLogger := log.Logger
	log.Logger = zerolog.Nop()
	inspectErr := inspectLeague(t.ctx, slug)
	log.Logger = origLogger

	pw.Close()
	os.Stdout = origStdout
	<-done

	return buf.String(), inspectErr
}

// refreshState silently runs inspectLeague and shows the result in the state
// panel. Zerolog is suppressed so noise doesn't pollute the output log.
func (t *leagueTUI) refreshState() {
	slug := t.slugInput.GetText()
	if slug == "" {
		return
	}

	output, _ := t.captureInspect(slug)

	t.app.QueueUpdateDraw(func() {
		t.stateView.Clear()
		fmt.Fprint(t.stateView, output)
	})
}

// runInspect captures inspectLeague output and streams it to the output panel.
// Like runOp it enforces the single-op lock, but uses captureInspect so there
// is only ever one goroutine writing to the output panel at a time.
func (t *leagueTUI) runInspect() {
	if t.running {
		fmt.Fprintln(t.outputView, "[red]⚠  busy — wait for current operation to finish[-]")
		return
	}
	slug := t.slugInput.GetText()
	t.running = true
	// Direct updates — main goroutine context.
	t.statusBar.SetText(" [yellow]⏳ Full Inspect...[-]  |  Tab=cycle focus  q=quit  ESC=close modal")
	fmt.Fprintf(t.outputView, "\n[teal]══ Full Inspect ══[-]  [gray]@ %s[-]\n",
		t.virtTime.Format("2006-01-02 15:04 UTC"))

	go func() {
		output, err := t.captureInspect(slug)
		t.app.QueueUpdateDraw(func() {
			if err != nil {
				fmt.Fprintf(t.outputView, "[red]✗ ERROR: %v[-]\n", err)
			} else {
				fmt.Fprint(t.outputView, output)
				fmt.Fprintln(t.outputView, "[green]✓ Done[-]")
			}
		})
		t.running = false
		t.setStatus(t.statusReady())
		go t.refreshState()
	}()
}

// ── Date modal ────────────────────────────────────────────────────────────────

func (t *leagueTUI) showDateModal() {
	input := tview.NewInputField()
	input.SetLabel("Datetime (RFC3339): ")
	input.SetText(t.virtTime.Format(time.RFC3339))
	input.SetFieldWidth(30)

	apply := func() {
		parsed, err := time.Parse(time.RFC3339, input.GetText())
		if err == nil {
			t.mu.Lock()
			t.virtTime = parsed.UTC()
			ts := t.virtTime.Format(time.RFC3339)
			display := t.formatTimeDisplay()
			t.mu.Unlock()

			os.Setenv("LEAGUE_NOW", ts)
			t.timeView.SetText(display) // handler context — direct update
		} else {
			fmt.Fprintf(t.outputView, "[red]bad date: %v[-]\n", err)
		}
		t.pages.RemovePage("date-modal")
		t.app.SetFocus(t.list)
	}

	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			apply()
		case tcell.KeyEscape:
			t.pages.RemovePage("date-modal")
			t.app.SetFocus(t.list)
		}
	})

	form := tview.NewForm()
	form.AddFormItem(input)
	form.AddButton("Set", apply)
	form.AddButton("Cancel", func() {
		t.pages.RemovePage("date-modal")
		t.app.SetFocus(t.list)
	})
	form.SetBorder(true)
	form.SetTitle(" Jump to Date ")
	form.SetTitleColor(tcell.ColorYellow)

	// Centre the form in the screen
	inner := tview.NewFlex()
	inner.SetDirection(tview.FlexRow)
	inner.AddItem(tview.NewBox(), 0, 1, false)
	inner.AddItem(form, 8, 0, true)
	inner.AddItem(tview.NewBox(), 0, 1, false)

	modal := tview.NewFlex()
	modal.AddItem(tview.NewBox(), 0, 1, false)
	modal.AddItem(inner, 64, 0, true)
	modal.AddItem(tview.NewBox(), 0, 1, false)

	t.pages.AddPage("date-modal", modal, true, true)
	t.app.SetFocus(form)
}

// ── Key handling ──────────────────────────────────────────────────────────────

func (t *leagueTUI) handleGlobalKeys(event *tcell.EventKey) *tcell.EventKey {
	// ESC always closes an open modal first.
	if event.Key() == tcell.KeyEscape {
		if t.pages.HasPage("date-modal") {
			t.pages.RemovePage("date-modal")
			t.app.SetFocus(t.list)
			return nil
		}
		return event
	}

	if t.pages.HasPage("date-modal") {
		return event
	}

	// Tab cycles focus between slug input, season input, and the action list.
	if event.Key() == tcell.KeyTab {
		t.focusIdx = (t.focusIdx + 1) % len(t.focusables)
		t.app.SetFocus(t.focusables[t.focusIdx])
		return nil
	}

	// 'q' quits only when the list (not a text input) is focused.
	if event.Rune() == 'q' || event.Rune() == 'Q' {
		if t.focusables[t.focusIdx] == t.list {
			t.app.Stop()
			return nil
		}
	}

	return event
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (t *leagueTUI) getSeason() int32 {
	n, err := strconv.Atoi(t.seasonInput.GetText())
	if err != nil || n <= 0 {
		return 1
	}
	return int32(n)
}

func (t *leagueTUI) usersFilePath() string {
	for _, p := range []string{
		"cmd/league-tester/test_users.json",
		"test_users.json",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "cmd/league-tester/test_users.json"
}

func (t *leagueTUI) formatTimeDisplay() string {
	return fmt.Sprintf("\n[yellow::b]%s[white::-]\n         UTC",
		t.virtTime.Format("2006-01-02  15:04"))
}

func (t *leagueTUI) setStatus(msg string) {
	t.app.QueueUpdateDraw(func() {
		t.statusBar.SetText(" " + msg + "  |  Tab=cycle focus  q=quit  ESC=close modal")
	})
}

func (t *leagueTUI) statusReady() string {
	return " [green]Ready[-]  |  Tab=focus  q=quit  ESC=modal  [gray]debug: /tmp/league-tester-debug.log[-]"
}
