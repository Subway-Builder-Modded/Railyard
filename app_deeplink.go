package main

import (
	"sync"

	"railyard/internal/deeplink"
	"railyard/internal/types"

	"github.com/wailsapp/wails/v2/pkg/options"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type deepLinkQueue struct {
	mu      sync.Mutex
	pending []deeplink.Target
}

func (q *deepLinkQueue) enqueue(target deeplink.Target) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pending = append(q.pending, target)
}

func (q *deepLinkQueue) drain() []deeplink.Target {
	q.mu.Lock()
	defer q.mu.Unlock()

	queued := append([]deeplink.Target(nil), q.pending...)
	q.pending = nil
	return queued
}

func (q *deepLinkQueue) consume() (deeplink.Target, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.pending) == 0 {
		return deeplink.Target{}, false
	}

	target := q.pending[0]
	q.pending = q.pending[1:]
	return target, true
}

func (a *App) onURLOpen(rawURL string) {
	a.HandleDeepLinkURL(rawURL)
}

// onSecondInstanceLaunch runs in the primary instance after Wails aborts a
// duplicate launch attempt. It forwards any deep-link args from the rejected
// process into the already-running app.
func (a *App) onSecondInstanceLaunch(data options.SecondInstanceData) {
	if target, ok := deeplink.ParseArgs(data.Args); ok {
		a.HandleDeepLinkTarget(target)
	}
}

func (a *App) HandleDeepLinkURL(rawURL string) {
	target, ok := deeplink.ParseURL(rawURL)
	if !ok {
		return
	}
	a.HandleDeepLinkTarget(target)
}

func (a *App) HandleDeepLinkTarget(target deeplink.Target) {
	if !target.Valid() {
		return
	}

	a.deepLinks.enqueue(target)
	if a.ctx != nil && a.IsStartupReady().Ready {
		a.emitPendingDeepLinks()
	}
}

func (a *App) emitPendingDeepLinks() {
	if a.ctx == nil {
		return
	}

	queued := a.deepLinks.drain()
	if len(queued) == 0 {
		return
	}

	a.bringToFront()

	for _, target := range queued {
		if target.Type == "GameStart" {
			wailsruntime.EventsEmit(a.ctx, "deeplink:start-game")
			continue
		}
		wailsruntime.EventsEmit(a.ctx, "deeplink:open", map[string]string{
			"type": target.Type,
			"id":   target.ID,
		})
	}
}

func (a *App) ConsumePendingDeepLink() types.DeepLinkResponse {
	if a.Logger != nil {
		a.Logger.Info("Consuming pending deep link")
	}
	target, ok := a.deepLinks.consume()
	if !ok {
		if a.Logger != nil {
			a.Logger.Warn("No pending deep link to consume")
		}
		return types.DeepLinkResponse{
			GenericResponse: types.SuccessResponse("No pending deep link"),
			Target:          nil,
		}
	}
	a.bringToFront()

	if a.Logger != nil {
		a.Logger.Info("Pending deep link consumed", "type", target.Type, "id", target.ID)
	}
	return types.DeepLinkResponse{
		GenericResponse: types.SuccessResponse("Pending deep link resolved"),
		Target: &types.DeepLinkTarget{
			Type: target.Type,
			ID:   target.ID,
		},
	}
}

func (a *App) bringToFront() {
	if a.ctx == nil {
		return
	}

	wailsruntime.WindowUnminimise(a.ctx)
	wailsruntime.WindowShow(a.ctx)
	wailsruntime.WindowSetAlwaysOnTop(a.ctx, true)
	wailsruntime.WindowSetAlwaysOnTop(a.ctx, false)
}
