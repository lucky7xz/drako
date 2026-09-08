package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/paths"
	"github.com/lucky7xz/drako/internal/profiles"
)

// This file owns the spec-function picker end to end: what a spec function is,
// which ones the specs directory offers, what each would do to the two staged
// lists, the keys that drive it, and how it draws. inventory_view.go and
// inventory_update.go know only two things about it — that tab opens it, and
// that a non-nil inventoryModel.specs floats over the inventory.
//
// It is one file rather than the repo's usual update/view pair because the
// picker is one concern. Splitting it by phase would put the rule that a row is
// dim in one file and the fact that dim means Help's colour in another, which
// is the scattering this file exists to undo.
//
// The shape inside it is three layers, and they are kept apart on purpose:
//
//	specPlan      what a function would do, as fact — no wording, no layout
//	planVerdict   facts -> a sentence, and how loudly to say it
//	renderSpecsPopup   sentences and facts -> a box
//
// Nothing here moves a file. Every function stages an arrangement and leaves
// the disk to [ Apply Changes ], which is what makes the whole feature a
// rearrangement of state the inventory already had.

// specVerb is one of the three things a spec file can be used for. They are
// not three operations: each one only computes a different argument for
// core.InventoryState.Arrange, and the arrangement is staged either way.
type specVerb int

const (
	specEquip specVerb = iota // equip exactly this spec, stash the rest
	specStash                 // stash this spec's decks, leave the rest alone
	specStrip                 // stash everything
)

// specPlan is one spec function, as fact: what it is and what it would do.
// Nothing here is a sentence and nothing here is a measurement — a plan reads
// the same whether it is about to be drawn, tested, or run.
//
// Everything the picker says about a row is derived from these fields rather
// than stored beside them, so the fact can always be recovered. An earlier
// version kept the warning text in the struct; testing the rule then meant
// asserting on English, and rewording a message broke tests about policy.
type specPlan struct {
	verb     specVerb
	name     string   // spec name; empty for strip
	profiles []string // what the spec file asked for; nil for strip
	desired  []string // the arrangement it would stage, locked deck included
	missing  []string // profiles it names that are not installed
	parseErr error    // the spec file does not parse; the lists above are empty
}

// specsOverlay is the spec-function picker floating over the inventory. It is
// a pointer field on inventoryModel — nil when closed — the same shape the
// delete confirmation uses.
type specsOverlay struct {
	plans  []specPlan
	cursor int
}

// selected returns the plan under the cursor, if there is one.
func (ov *specsOverlay) selected() (specPlan, bool) {
	if ov == nil || ov.cursor < 0 || ov.cursor >= len(ov.plans) {
		return specPlan{}, false
	}
	return ov.plans[ov.cursor], true
}

// subject names the plan for a sentence about it.
func (p specPlan) subject() string {
	if p.verb == specStrip {
		return "strip"
	}
	return "spec '" + p.name + "'"
}

// label is the row's left column.
func (p specPlan) label() string {
	if p.verb == specStrip {
		return "strip"
	}
	return p.name
}

// detail is the row's right column: what this plan is made of.
func (p specPlan) detail() string {
	switch {
	case p.verb == specStrip:
		return "stash every equipped deck"
	case p.parseErr != nil:
		return "unreadable"
	}
	return strings.Join(p.profiles, " ")
}

// specVerbAction is the bare verb, in the imperative. The popup's key hint
// shows it for the selected row: the group header can scroll off a long spec
// list, and "which verb am I about to fire?" must never be a question. Just the
// verb — the spec's name is on the highlighted row already, and putting it here
// too would make the hint's length depend on the name.
func specVerbAction(v specVerb) string {
	switch v {
	case specEquip:
		return "equip"
	case specStash:
		return "stash"
	default:
		return "strip"
	}
}

// action names what this plan did, for the status line after it runs — where
// there is room for the spec's name and no name left on screen to read it from.
func (p specPlan) action() string {
	if p.verb == specStrip {
		return "strip"
	}
	return specVerbAction(p.verb) + " " + p.subject()
}

// specVerbHeader names the group a plan belongs to. The overlay draws a header
// wherever the verb changes, so headers never become rows the cursor can land
// on and the cursor arithmetic stays a plain index.
func specVerbHeader(v specVerb) string {
	switch v {
	case specEquip:
		return "EQUIP"
	case specStash:
		return "STASH"
	default:
		return "STRIP"
	}
}

// lockedFile returns the equipped file the pivot lock points at, if any. The
// locked deck is the one thing no spec function may move: the inventory
// already refuses to lift it by hand, and a bulk function must not be the way
// around that guard.
func (m Model) lockedFile() string {
	if m.profile.pivotName == "" {
		return ""
	}
	for _, f := range m.inventory.State.Visible {
		if m.isLockedProfile(strings.TrimSuffix(f, profiles.ProfileSuffix)) {
			return f
		}
	}
	return ""
}

// buildSpecPlans reads the specs directory once and turns every spec into its
// equip and stash plans, followed by the single strip plan. Strip is offered
// even with no specs on disk: it needs no spec file.
//
// It decides nothing about what is allowed and says nothing in English. Those
// are planVerdict's job, from the facts recorded here.
func (m Model) buildSpecPlans() []specPlan {
	specs, err := config.DiscoverSpecs(paths.SpecsDir(m.profile.configDir))
	if err != nil {
		// A directory we cannot read yields no specs; strip still works.
		specs = nil
	}
	state := m.inventory.State

	// Every file the inventory knows about, by normalized name, so a spec's
	// bare "git" finds "git.profile.toml" wherever it currently sits.
	byName := make(map[string]string, len(state.Visible)+len(state.Inventory))
	for _, f := range append(append([]string{}, state.Visible...), state.Inventory...) {
		byName[profiles.NormalizeName(strings.TrimSuffix(f, profiles.ProfileSuffix))] = f
	}

	var equipPlans, stashPlans []specPlan
	for _, s := range specs {
		if s.Err != nil {
			equipPlans = append(equipPlans, specPlan{verb: specEquip, name: s.Name, parseErr: s.Err})
			stashPlans = append(stashPlans, specPlan{verb: specStash, name: s.Name, parseErr: s.Err})
			continue
		}

		files, missing := resolveSpec(s.Profiles, byName)
		equipPlans = append(equipPlans, specPlan{
			verb: specEquip, name: s.Name, profiles: s.Profiles,
			desired: files, missing: missing,
		})
		stashPlans = append(stashPlans, specPlan{
			verb: specStash, name: s.Name, profiles: s.Profiles,
			desired: subtract(state.Visible, files),
		})
	}

	plans := append(equipPlans, stashPlans...)
	plans = append(plans, specPlan{verb: specStrip})

	// The locked deck is kept by every plan, applied here in one pass rather
	// than at each construction site. An invariant that depends on remembering
	// to call a helper is one a fourth verb can quietly break.
	if locked := m.lockedFile(); locked != "" {
		for i := range plans {
			plans[i].desired = withLocked(locked, plans[i].desired)
		}
	}
	return plans
}

// resolveSpec maps a spec's bare profile names onto the files that exist,
// reporting the names that match nothing.
func resolveSpec(want []string, byName map[string]string) (files, missing []string) {
	for _, p := range want {
		if f, ok := byName[profiles.NormalizeName(p)]; ok {
			files = append(files, f)
		} else {
			missing = append(missing, p)
		}
	}
	return files, missing
}

// withLocked puts the locked deck at the head of an arrangement, without
// duplicating it if the plan already asked for it.
func withLocked(locked string, files []string) []string {
	out := []string{locked}
	for _, f := range files {
		if f != locked {
			out = append(out, f)
		}
	}
	return out
}

// subtract returns the files in from that are not in remove.
func subtract(from, remove []string) []string {
	drop := make(map[string]bool, len(remove))
	for _, f := range remove {
		drop[f] = true
	}
	var out []string
	for _, f := range from {
		if !drop[f] {
			out = append(out, f)
		}
	}
	return out
}

// namesInAMessage is how many names a sentence about them may list before it
// starts counting instead. A spec missing twenty decks has to stay one
// readable sentence, not a wall of names.
const namesInAMessage = 3

// summarizeNames lists a few names and counts the rest.
func summarizeNames(names []string) string {
	if len(names) <= namesInAMessage {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s and %d more",
		strings.Join(names[:namesInAMessage], ", "), len(names)-namesInAMessage)
}

// verdictKind is how loudly a verdict is said, and — for blocked — whether the
// plan may run at all.
type verdictKind int

const (
	verdictNone    verdictKind = iota
	verdictInfo                // neutral: what the row could not fit
	verdictWarn                // it will run, but skip something
	verdictBlocked             // it cannot run
)

// planVerdict turns a plan's facts into the sentence the picker shows, in
// priority order: cannot-parse, cannot-arrange, will-skip. Everything the
// picker refuses or warns about is decided here, once, so the dimming, the
// key hint and the enter handler cannot disagree.
//
// Whether the arrangement fits is asked of core rather than recomputed: the cap
// is core's rule, and a second copy of it here would be a rule that can drift
// from the one Apply actually enforces.
func (m Model) planVerdict(p specPlan) (string, verdictKind) {
	switch {
	case p.parseErr != nil:
		return fmt.Sprintf("%s does not parse: %v", p.subject(), p.parseErr), verdictBlocked

	case m.inventory.State.CanArrange(p.desired) != nil:
		return fmt.Sprintf("%s: %v", p.subject(), m.inventory.State.CanArrange(p.desired)), verdictBlocked

	case len(p.missing) > 0:
		// Not a refusal. Reconcile ignores names it has no file for, and
		// "drako spec" equips the rest without comment — so the picker equips
		// the rest too, and says out loud what it skipped.
		return fmt.Sprintf("%s skips %s — not installed",
			p.subject(), summarizeNames(p.missing)), verdictWarn
	}
	return "", verdictNone
}

// updateSpecsOverlay drives the picker. It owns every key while open, so the
// inventory's own bindings can't fire underneath it.
func (m Model) updateSpecsOverlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	inv := &m.inventory
	ov := inv.specs

	switch {
	case Matches(m.Config.Keys, msg, "ctrl+c"):
		m.Quitting = true
		return m, tea.Quit

	// Tab both opens and closes: the overlay is a place you step into and
	// back out of, not a mode you have to remember how to leave.
	case IsPathGridMode(m.Config.Keys, msg), IsCancel(m.Config.Keys, msg):
		inv.specs = nil

	case IsUp(m.Config.Keys, msg):
		if ov.cursor > 0 {
			ov.cursor--
		}
	case IsDown(m.Config.Keys, msg):
		if ov.cursor < len(ov.plans)-1 {
			ov.cursor++
		}

	case IsConfirm(m.Config.Keys, msg):
		plan, ok := ov.selected()
		if !ok {
			return m, nil
		}
		// A blocked plan does nothing. Its reason is already on screen — the
		// popup shows the selected row's verdict as you move onto it, so this
		// is not a silent no-op, it is a key with nothing left to say.
		if _, kind := m.planVerdict(plan); kind == verdictBlocked {
			return m, nil
		}
		if err := inv.State.Arrange(plan.desired); err != nil {
			// Unreachable: planVerdict asked CanArrange the same question, and
			// nothing can change the state while the picker holds every key.
			inv.specs, inv.status, inv.statusOK = nil, "Could not stage: "+err.Error(), false
			return m, nil
		}
		inv.specs = nil
		inv.status, inv.statusOK = m.stagedSummary(plan), true
		// Land on Apply, which is the only thing left to do. The arrangement
		// behind the popup is already what the spec asked for; walking the
		// lists to reach the button would only be a chance to disturb it.
		inv.focusedList, inv.cursor = focusApply, 0
	}

	return m, nil
}

// stagedSummary says what was staged and what it cost, including the things the
// plan did not do: skip a deck it could not find, move the locked one.
//
// What happened, then what it did not do, then what to press. The line is
// truncated to the terminal width, so the exceptions come before the reminder —
// the button they point at is on screen two rows up anyway.
func (m Model) stagedSummary(p specPlan) string {
	msg := fmt.Sprintf("Staged: %s — %d equipped, %d stashed.",
		p.action(), len(m.inventory.State.Visible), len(m.inventory.State.Inventory))

	if len(p.missing) > 0 {
		msg += fmt.Sprintf(" Skipped %s (not installed).", summarizeNames(p.missing))
	}
	if locked := m.lockedFile(); locked != "" {
		msg += fmt.Sprintf(" Kept %s (locked).", strings.TrimSuffix(locked, profiles.ProfileSuffix))
	}
	return msg + " Apply to commit."
}

// The picker's box is a fixed size, not one derived from what it happens to
// contain. Sizing it to its content means every keystroke that changes a
// message — moving onto a spec with a longer name, or one carrying a warning —
// resizes the border around it, and the box appears to shake as you navigate.
//
// specPopupRows is how many rows the box spends on everything that is not a
// spec line: DropdownPopup's border, padding and margin (2 each), the title and
// the blank under it, the blank above the footer, the two reserved verdict
// lines, and the key hints.
const (
	specPopupChrome  = 12
	specPopupWidth   = 60 // content columns, before border/padding
	specPopupFloor   = 8  // last resort, so something still renders
	specVerdictLines = 2  // reserved, so a wrapped warning cannot change the height
)

// noVerb is the "no group drawn yet" sentinel for the header loop below.
const noVerb = specVerb(-1)

// specContentWidth is the box's inner width: fixed, and only ever narrowed to
// fit a small terminal. It does not depend on the rows, so the border stays put
// while the cursor moves.
//
// The floor is a last resort, applied only after the terminal has had its say.
// A minimum that outranks the available width is not a minimum, it is an
// overflow: the box would claim room the terminal does not have and lose its
// right border off the edge of the screen.
func specContentWidth(termWidth int) int {
	fits := termWidth - LayoutSideMargin - 6 // border, padding and margin
	return max(specPopupFloor, min(specPopupWidth, fits))
}

// specDetailFit shrinks a space-separated deck list to width by dropping whole
// names and counting what it dropped: "alpha bravo charlie +6". A plain
// truncation would cut mid-name ("foxtr...") and, worse, hide how much is
// hidden — the reader can't tell a spec of four decks from one of twelve. The
// full list is still one keypress away: it fills the reserved lines below when
// the row is selected.
func specDetailFit(detail string, width int) string {
	if width <= 0 || lipgloss.Width(detail) <= width {
		return detail
	}
	names := strings.Fields(detail)
	if len(names) < 2 {
		return truncateText(detail, width)
	}

	used, shown := 0, 0
	for i, n := range names {
		cost := lipgloss.Width(n)
		if i > 0 {
			cost++ // the separating space
		}
		tail := 0
		if rest := len(names) - i - 1; rest > 0 {
			tail = lipgloss.Width(fmt.Sprintf(" +%d", rest))
		}
		if used+cost+tail > width {
			break
		}
		used, shown = used+cost, shown+1
	}

	// Too narrow for even one name and its counter: give the count alone.
	if shown == 0 {
		return truncateText(fmt.Sprintf("%d decks", len(names)), width)
	}
	return fmt.Sprintf("%s +%d", strings.Join(names[:shown], " "), len(names)-shown)
}

// renderSpecsPopup draws the spec-function picker. Group headers are derived
// from where the verb changes rather than stored as rows, so every row in the
// overlay is one the cursor can land on and selection stays a plain index.
//
// Like the other popups, every segment renders on the popup background: the
// reset that ends each nested Render would otherwise punch holes in it.
func (m Model) renderSpecsPopup() string {
	ov := m.inventory.specs
	bg := m.styles.DropdownPopup.GetBackground()
	bgFill := lipgloss.NewStyle().Background(bg)
	title := m.styles.Title.Background(bg)
	header := m.styles.ListHeader.Background(bg)
	item := m.styles.Item.Background(bg)
	selected := m.styles.SelectedItem.Background(bg)
	// Dim borrows Help's colour but keeps Item's padding: a blocked row has to
	// line up with the rows around it, or the list reads as ragged.
	dim := item.Foreground(m.styles.Help.GetForeground())
	cursor := m.styles.SelectedCursor.Background(bg)

	width := specContentWidth(m.termWidth)
	// A row is "  " gutter + Item's one column of padding each side.
	rowText := width - 4

	// Size the name column to the widest label so the details line up — but
	// never past half the row, or one unusually long spec name would truncate
	// every other row's profile list. The cap only bites in a narrow box, which
	// is exactly where the space is worth arguing over.
	labelW := 0
	for _, p := range ov.plans {
		labelW = max(labelW, lipgloss.Width(p.label()))
	}
	labelW = min(labelW, rowText/2)

	// Ask each plan's verdict once, here. The row's colour, its ⚠, the key
	// hint and the lines at the bottom are four views of the same answer, and
	// asking four times is how they would come to disagree.
	verdicts := make([]string, len(ov.plans))
	kinds := make([]verdictKind, len(ov.plans))
	fitted := make([]string, len(ov.plans))
	detailW := rowText - labelW - 2
	for i, p := range ov.plans {
		verdicts[i], kinds[i] = m.planVerdict(p)
		fitted[i] = specDetailFit(p.detail(), detailW)
	}

	// Build every display line, remembering where the cursor's line landed so
	// the window can centre on it even though headers shift the indices.
	// The leading gutter is rendered rather than written bare: a plain string
	// would carry the terminal's own background and stripe the box.
	var lines []string
	cursorLine := 0
	lastVerb := noVerb
	for i, p := range ov.plans {
		if p.verb != lastVerb {
			if lastVerb != noVerb {
				lines = append(lines, "")
			}
			lines = append(lines, bgFill.Render("  ")+header.Render(specVerbHeader(p.verb)))
			lastVerb = p.verb
		}

		style := item
		switch {
		case i == ov.cursor:
			style = selected
		case kinds[i] == verdictBlocked:
			style = dim
		}

		detail := fitted[i]
		if kinds[i] == verdictWarn {
			// The lines below name what is missing; this only has to say that
			// something is, so the row is worth reading before choosing.
			detail += "  ⚠"
		}
		// Truncate the text, never the rendered line: cutting a styled string
		// would slice through the escape sequence that ends it.
		body := truncateText(fmt.Sprintf("%-*s  %s", labelW, truncateText(p.label(), labelW), detail), rowText)
		mark := bgFill.Render("  ")
		if i == ov.cursor {
			mark = cursor.Render("► ")
			cursorLine = len(lines)
		}
		lines = append(lines, mark+style.Render(body))
	}

	// Window the lines when the popup would outgrow the terminal.
	budget := m.termHeight - specPopupChrome
	if visible := visibleCount(budget, 1, len(lines), 1); visible < len(lines) {
		win := window(cursorLine, len(lines), visible)
		lines = append([]string{}, lines[win.start:win.end]...)
		down, up := " ", " "
		if win.hiddenAfter > 0 {
			down = "▾"
		}
		if win.hiddenBefore > 0 {
			up = "▴"
		}
		lines = append(lines, cursor.Render(down+" "+up))
	}

	// Name the verb enter will take, so a scrolled-off group header can never
	// leave it in doubt — and offer no action at all on a row that has none,
	// rather than promising one enter would not perform. The verb, not the
	// spec's name: the name is on the highlighted row already, and a long one
	// would push "tab/esc: back" off the end of the line.
	keys := "↑/↓: pick  tab/esc: back"
	if _, ok := ov.selected(); ok && kinds[ov.cursor] != verdictBlocked {
		keys = "enter: " + specVerbAction(ov.plans[ov.cursor].verb) + "  " + keys
	}
	hints := dim.Render(truncateText(keys, rowText))

	// The reserved lines describe the selected row: its verdict if it has one,
	// otherwise the decks it names in full — the row itself only has space for
	// a preview. Shown as the cursor lands, not once a key is pressed against
	// it, and always specVerdictLines tall so nothing above can shift.
	//
	// A verdict always wins over the listing. Whether the listing is worth
	// showing is the one question only the view can answer, since it turns on
	// what actually fit on the row.
	text, kind := "", verdictNone
	if p, ok := ov.selected(); ok {
		text, kind = verdicts[ov.cursor], kinds[ov.cursor]
		if kind == verdictNone && p.verb != specStrip && fitted[ov.cursor] != p.detail() {
			text, kind = p.detail(), verdictInfo
		}
	}

	// Red for "this cannot run", amber for "this will skip something", the
	// quiet Help colour for a plain listing. All three keep Item's padding, so
	// they line up with the rows.
	tone := dim
	switch kind {
	case verdictBlocked:
		tone = item.Foreground(m.styles.StatusNegative.GetForeground())
	case verdictWarn:
		tone = item.Foreground(warnAmber)
	}

	verdict := make([]string, specVerdictLines)
	for i := range verdict {
		verdict[i] = bgFill.Render(" ")
	}
	wrapped := WrapText(text, rowText)
	for i, ln := range wrapped {
		if i >= specVerdictLines {
			// More than fits: mark the last kept line so the cut is visible
			// rather than silently swallowing the tail.
			verdict[specVerdictLines-1] = bgFill.Render("  ") +
				tone.Render(truncateText(wrapped[specVerdictLines-1]+" "+strings.Join(wrapped[i:], " "), rowText))
			break
		}
		verdict[i] = bgFill.Render("  ") + tone.Render(ln)
	}

	// The title is truncated like every other line: one string left out of the
	// width discipline is enough to push the border off a narrow screen.
	raw := append([]string{bgFill.Render("  ") + title.Render(truncateText("Spec functions", rowText)), ""}, lines...)
	raw = append(raw, "")
	raw = append(raw, verdict...)
	raw = append(raw, bgFill.Render("  ")+hints)

	return m.styles.DropdownPopup.Render(
		lipgloss.JoinVertical(lipgloss.Left, padLinesTo(raw, width, bgFill)...),
	)
}
