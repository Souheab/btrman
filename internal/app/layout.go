package app

const (
	sidebarThreshold = 90
	sidebarWidth     = 24
)

func (m Model) previewWidth() int {
	if m.width < sidebarThreshold {
		return max(60, m.width-8)
	}
	return max(60, (m.width*3)/5-8)
}

func (m *Model) resize() {
	if m.width <= 0 {
		m.width = 100
	}
	if m.height <= 0 {
		m.height = 30
	}
	m.pageInput.Width = max(20, m.width-10)
	m.findInput.Width = max(20, m.width-10)
	m.swiperInput.Width = max(20, m.width-10)
	m.viewport.Width = max(20, m.contentWidth())
	m.viewport.Height = max(5, m.height-4)
	m.rebuildViewportContent()
}

func (m Model) contentWidth() int {
	if m.width >= sidebarThreshold {
		return m.width - sidebarWidth - 2
	}
	return m.width
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp(value, minValue, maxValue int) int {
	if maxValue < minValue {
		return minValue
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
