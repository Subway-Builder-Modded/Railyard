package main

import (
	"testing"

	"railyard/internal/deeplink"
	"railyard/internal/logger"

	"github.com/stretchr/testify/require"
	"github.com/wailsapp/wails/v2/pkg/options"
)

func TestHandleDeepLinkTargetQueuesPendingLink(t *testing.T) {
	app := &App{Logger: logger.LoggerAtPath("")}

	app.HandleDeepLinkTarget(deeplink.Target{Type: "maps", ID: "amsterdam"})

	response := app.ConsumePendingDeepLink()
	require.Equal(t, "maps", response.Target.Type)
	require.Equal(t, "amsterdam", response.Target.ID)
	require.Nil(t, app.ConsumePendingDeepLink().Target)
}

func TestHandleDeepLinkTargetIgnoresInvalidTargets(t *testing.T) {
	app := &App{Logger: logger.LoggerAtPath("")}

	app.HandleDeepLinkTarget(deeplink.Target{Type: "invalid", ID: "nope"})

	require.Nil(t, app.ConsumePendingDeepLink().Target)
}

func TestOnSecondInstanceLaunchQueuesDeepLinkFromArgs(t *testing.T) {
	app := &App{Logger: logger.LoggerAtPath("")}

	app.onSecondInstanceLaunch(options.SecondInstanceData{
		Args: []string{"railyard://open?type=mods&id=signal-pack"},
	})

	response := app.ConsumePendingDeepLink()
	require.Equal(t, "mods", response.Target.Type)
	require.Equal(t, "signal-pack", response.Target.ID)
}

func TestEmitPendingDeepLinksDoesNotDrainWithoutContext(t *testing.T) {
	app := &App{Logger: logger.LoggerAtPath("")}
	app.HandleDeepLinkTarget(deeplink.Target{Type: "mods", ID: "signal-pack"})

	app.emitPendingDeepLinks()

	response := app.ConsumePendingDeepLink()
	require.NotNil(t, response.Target)
	require.Equal(t, "mods", response.Target.Type)
	require.Equal(t, "signal-pack", response.Target.ID)
}
