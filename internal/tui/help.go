package tui

import (
	"charm.land/bubbles/v2/key"
)

// keyMap implements help.KeyMap
type keyMap struct {
	mode ViewMode
}

func (km keyMap) FullHelp() [][]key.Binding {
	lastGroup := []key.Binding{
		enterScrollKey,
		exitScrollKey,
	}
	if km.mode != ModeRepo {
		lastGroup = append(lastGroup, modeKey)
	}
	lastGroup = append(lastGroup, backKey, quitKey, helpKey)

	return [][]key.Binding{
		{
			nextRowKey,
			prevRowKey,
			nextPaneKey,
			prevPaneKey,
			gotoTopKey,
			gotoBottomKey,
			zoomPaneKey,
		},
		{
			searchKey,
			cancelSearchKey,
			applySearchKey,
			nextSearchMatchKey,
			prevSearchMatchKey,
		},
		{
			rerunKey,
			openUrlKey,
			openPRKey,
			refreshAllKey,
		},
		lastGroup,
	}
}

func (km keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		helpKey,
	}
}

var keys = keyMap{}
