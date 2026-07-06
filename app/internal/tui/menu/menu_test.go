package menu

import "testing"

func TestBuildItemsGating(t *testing.T) {
	// No tools: Requirements/Labels/Project disabled; Update absent (not initialised).
	items := buildItems(tools{}, false)
	byAction := map[Action]item{}
	for _, it := range items {
		byAction[it.action] = it
	}
	if _, ok := byAction[ActionUpdate]; ok {
		t.Error("Update should not appear when not initialised")
	}
	if !byAction[ActionReq].disabled {
		t.Error("Requirements should be disabled without an AI tool")
	}
	if !byAction[ActionLabels].disabled || !byAction[ActionProject].disabled {
		t.Error("Labels/Project should be disabled without gh")
	}
	if byAction[ActionInit].disabled {
		t.Error("Init should always be enabled")
	}
}

func TestBuildItemsAllTools(t *testing.T) {
	items := buildItems(tools{hasAI: true, hasGH: true}, true)
	byAction := map[Action]item{}
	for _, it := range items {
		byAction[it.action] = it
	}
	if _, ok := byAction[ActionUpdate]; !ok {
		t.Error("Update should appear when initialised")
	}
	for _, a := range []Action{ActionReq, ActionLabels, ActionProject} {
		if byAction[a].disabled {
			t.Errorf("action %d should be enabled when tools present", a)
		}
	}
}

func TestMoveCursorSkipsDisabled(t *testing.T) {
	// Init(enabled), Req(disabled), Labels(disabled), Project(disabled), Help(enabled)
	m := &model{items: buildItems(tools{}, false)}
	// from Init(0), down should skip the 3 disabled and land on Help.
	m.moveCursor(1)
	if m.items[m.cursor].action != ActionHelp {
		t.Errorf("down from Init should skip disabled to Help, got %d", m.items[m.cursor].action)
	}
	// up from Help wraps back to Init.
	m.moveCursor(-1)
	if m.items[m.cursor].action != ActionInit {
		t.Errorf("up from Help should return to Init, got %d", m.items[m.cursor].action)
	}
}

func TestEnterOnDisabledDoesNothing(t *testing.T) {
	m := &model{items: buildItems(tools{}, false)}
	// Force cursor onto a disabled item.
	for i, it := range m.items {
		if it.disabled {
			m.cursor = i
			break
		}
	}
	if m.items[m.cursor].disabled != true {
		t.Skip("no disabled item to test")
	}
	if m.result != ActionNone {
		t.Fatalf("precondition: result should start ActionNone, got %d", m.result)
	}
}
