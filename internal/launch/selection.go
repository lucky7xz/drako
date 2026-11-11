package launch

import "github.com/lucky7xz/drako/internal/profile"

// Selection is what the user picked: the cell named Cell, or item Item (1-based)
// of its dropdown. Item names alone can repeat across folders.
type Selection struct {
	Cell string
	Item int // 0 selects the cell itself
}

// Resolve finds the cell among cells, a profile's top-level commands (drako
// check keeps their names unique), and, for Item > 0, the item inside it.
func (s Selection) Resolve(cells []profile.Command) (cell *profile.Command, item *profile.CommandItem, ok bool) {
	for i := range cells {
		c := &cells[i]
		if c.Name != s.Cell {
			continue
		}
		switch {
		case s.Item == 0:
			return c, nil, true
		case s.Item > 0 && s.Item <= len(c.Items):
			return c, &c.Items[s.Item-1], true
		}
		return nil, nil, false
	}
	return nil, nil, false
}

// Name is the picked entry's name as shown on screen: the item's, else the cell's.
func (s Selection) Name(cells []profile.Command) string {
	if _, item, ok := s.Resolve(cells); ok && item != nil {
		return item.Name
	}
	return s.Cell
}
