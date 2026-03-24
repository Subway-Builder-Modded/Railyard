package deeplink

import (
	"net/url"
	"strings"
)

type Target struct {
	Type string
	ID   string
}

func (t Target) Valid() bool {
	if t.ID == "" {
		return false
	}
	return t.Type == "mods" || t.Type == "maps" || t.Type == "GameStart"
}

func ParseURL(raw string) (Target, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return Target{}, false
	}
	if !strings.EqualFold(parsed.Scheme, "railyard") {
		return Target{}, false
	}

	isOpenAction := strings.EqualFold(parsed.Host, "open") || strings.EqualFold(strings.Trim(parsed.Path, "/"), "open")
	if !isOpenAction {
		isStartGameAction := strings.EqualFold(parsed.Host, "start-game") || strings.EqualFold(strings.Trim(parsed.Path, "/"), "start-game")
		if !isStartGameAction {
			return Target{}, false
		} else {
			return Target{
				Type: "GameStart",
			}, true // Start game action doesn't require a valid target, return empty target with true to indicate the action is recognized
		}
	}

	target := Target{
		Type: strings.ToLower(strings.TrimSpace(parsed.Query().Get("type"))),
		ID:   strings.TrimSpace(parsed.Query().Get("id")),
	}
	if !target.Valid() {
		return Target{}, false
	}
	return target, true
}

func ParseArgs(args []string) (Target, bool) {
	for _, arg := range args {
		target, ok := ParseURL(arg)
		if ok {
			return target, true
		}
	}
	return Target{}, false
}
