package browser

// History manages a back/forward navigation stack.
type History struct {
	entries []string
	pos     int // current position in the stack
}

// NewHistory creates an empty navigation history.
func NewHistory() *History {
	return &History{
		entries: nil,
		pos:     -1,
	}
}

// Push adds a new URL to the history, truncating any forward entries.
func (h *History) Push(url string) {
	// If we're not at the end, truncate forward history.
	if h.pos < len(h.entries)-1 {
		h.entries = h.entries[:h.pos+1]
	}
	h.entries = append(h.entries, url)
	h.pos = len(h.entries) - 1
}

// Back moves one step back in history. Returns the URL and true if possible.
func (h *History) Back() (string, bool) {
	if h.pos <= 0 {
		return "", false
	}
	h.pos--
	return h.entries[h.pos], true
}

// Forward moves one step forward in history. Returns the URL and true if possible.
func (h *History) Forward() (string, bool) {
	if h.pos >= len(h.entries)-1 {
		return "", false
	}
	h.pos++
	return h.entries[h.pos], true
}

// Current returns the current URL, or empty string if history is empty.
func (h *History) Current() string {
	if h.pos < 0 || h.pos >= len(h.entries) {
		return ""
	}
	return h.entries[h.pos]
}

// CanGoBack reports whether there is a previous entry.
func (h *History) CanGoBack() bool {
	return h.pos > 0
}

// CanGoForward reports whether there is a next entry.
func (h *History) CanGoForward() bool {
	return h.pos < len(h.entries)-1
}

// Len returns the total number of entries.
func (h *History) Len() int {
	return len(h.entries)
}

// Clear resets the history.
func (h *History) Clear() {
	h.entries = nil
	h.pos = -1
}
