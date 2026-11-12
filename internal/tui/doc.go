// Package tui is drako's screen: one Bubble Tea program whose model holds the
// state, whose Update turns keys and messages into new state, and whose View
// draws it. It starts nothing and writes nothing itself: it asks lair for
// drako's folder, host for the machine and terminal, proc for the editor, and
// hands its host (app) an Outcome saying what to run once it is off screen.
//
// Where things are:
//
//	the loop      model.go (the state), update.go (dispatch), view.go,
//	              startup.go (creating the model, the Session between laps),
//	              outcome.go (Setup in, Outcome out), bundle.go (applying a
//	              loaded configuration), notices.go (popups drako raises
//	              itself), input.go, key_guards.go, leader.go, layout.go
//	grid          grid_update.go, grid_view.go, grid_nav.go
//	dropdown      dropdown_update.go, overlays_view.go
//	popups        info_update.go, overlays_view.go, detail.go, highlight.go,
//	              viewport.go, textwrap.go
//	inventory     inventory_update.go, inventory_view.go, inventory_specs.go
//	              (the spec picker), editor.go
//	path browser  path_mode.go
//	batch         batch_mode.go, batch_dialog.go, batch_view.go
//	key sheet     key_sheet.go
//	lock          lock.go, overlays_view.go
//	profiles      profile_state.go (active profile, switching, the pin)
//	look          styles.go, spinner.go, glassroot.go
//
// The screens are modes of the one program, not programs of their own: they
// share the keybindings, the styles, the terminal size and the active
// profile, which all sit on the model.
package tui
