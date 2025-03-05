package tui

// LayoutManager helps calculate component dimensions based on terminal size
type LayoutManager struct {
	Width  int
	Height int
}

// NewLayoutManager creates a new layout manager with the given dimensions
func NewLayoutManager(width, height int) *LayoutManager {
	return &LayoutManager{
		Width:  width,
		Height: height,
	}
}

// FilesListDimensions returns the dimensions for the files list
func (l *LayoutManager) FilesListDimensions() (width, height int) {
	return l.Width, l.Height * FilesViewHeight / 100
}

// ChangesViewDimensions returns the dimensions for the changes viewport
func (l *LayoutManager) ChangesViewDimensions() (width, height int) {
	return l.Width, l.Height * ChangesViewHeight / 100
}

// MessageInputDimensions returns the dimensions for the message input
func (l *LayoutManager) MessageInputDimensions() (width, height int) {
	return l.Width, l.Height * MessageViewHeight / 100
}

// ContentHeight returns the available height for content (excluding status line)
func (l *LayoutManager) ContentHeight() int {
	return l.Height - 2 // Reserve 2 lines for status
}
