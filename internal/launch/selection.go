package launch

import "github.com/lucky7xz/drako/internal/profile"

// Selection is what the user picked: command N of the profile (counted from 1,
// in the order the profile lists them), or item Item (1-based) of its
// dropdown. The zero Selection picks nothing. A cell is picked by where it
// stands, never by its name: two cells may share a name.
type Selection struct {
	N    int
	Item int // 0 selects the cell itself
}

// Picked reports whether the selection names a cell at all.
func (s Selection) Picked() bool { return s.N > 0 }

// Resolve finds the cell among cells, a profile's top-level commands, and,
// for Item > 0, the item inside it.
func (s Selection) Resolve(cells []profile.Command) (cell *profile.Command, item *profile.CommandItem, ok bool) {
	if s.N < 1 || s.N > len(cells) {
		return nil, nil, false
	}
	c := &cells[s.N-1]
	switch {
	case s.Item == 0:
		return c, nil, true
	case s.Item > 0 && s.Item <= len(c.Items):
		return c, &c.Items[s.Item-1], true
	}
	return nil, nil, false
}

// Name is the picked entry's name as shown on screen: the item's, else the
// cell's; "" when the selection resolves to nothing.
func (s Selection) Name(cells []profile.Command) string {
	cell, item, ok := s.Resolve(cells)
	switch {
	case !ok:
		return ""
	case item != nil:
		return item.Name
	}
	return cell.Name
}
