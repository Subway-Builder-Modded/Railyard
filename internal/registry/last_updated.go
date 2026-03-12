package registry

import (
	"fmt"
	"math"
	"sync"
	"time"

	"railyard/internal/types"
)

const lastUpdatedWorkerLimit = 6

type lastUpdatedArgs struct {
	id         string
	updateType string
	source     string
}

func (r *Registry) loadLastUpdated(
	mods []types.ModManifest,
	maps []types.MapManifest,
) (map[string]int64, map[string]int64) {
	r.logger.Info(
		"Resolving last updated metadata",
		"mod_count", len(mods),
		"map_count", len(maps),
	)

	mapSources := getLastUpdatedArgs(
		maps,
		func(manifest types.MapManifest) string { return manifest.ID },
		func(manifest types.MapManifest) types.UpdateConfig { return manifest.Update },
	)
	modSources := getLastUpdatedArgs(
		mods,
		func(manifest types.ModManifest) string { return manifest.ID },
		func(manifest types.ModManifest) types.UpdateConfig { return manifest.Update },
	)

	mapEntries, mapWarnings := r.resolveLastUpdated(mapSources)
	modEntries, modWarnings := r.resolveLastUpdated(modSources)

	if len(mapWarnings) > 0 {
		r.logger.Warn("Last updated metadata resolved with warnings", "asset_type", types.AssetTypeMap, "warning_count", len(mapWarnings))
	}
	if len(modWarnings) > 0 {
		r.logger.Warn("Last updated metadata resolved with warnings", "asset_type", types.AssetTypeMod, "warning_count", len(modWarnings))
	}

	r.logger.Info(
		"Resolved last updated metadata",
		"resolved_mods", len(modEntries),
		"resolved_maps", len(mapEntries),
	)

	return modEntries, mapEntries
}

func updateManifestLastUpdated(
	mods []types.ModManifest,
	maps []types.MapManifest,
	modLastUpdated map[string]int64,
	mapLastUpdated map[string]int64,
) {
	for i := range mods {
		mods[i].LastUpdated = modLastUpdated[mods[i].ID]
	}
	for i := range maps {
		maps[i].LastUpdated = mapLastUpdated[maps[i].ID]
	}
}

func (r *Registry) resolveLastUpdated(
	sources []lastUpdatedArgs,
) (map[string]int64, []error) {
	results := make(map[string]int64, len(sources))
	warnings := make([]error, 0)

	if len(sources) == 0 {
		return results, warnings
	}

	workerLimit := lastUpdatedWorkerLimit
	if len(sources) < workerLimit {
		workerLimit = len(sources)
	}

	sem := make(chan struct{}, workerLimit)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, source := range sources {
		current := source
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			versions, err := r.GetVersions(current.updateType, current.source)
			if err != nil {
				r.logger.Warn("Failed to fetch versions for last updated", "asset_id", current.id, "update_type", current.updateType, "source", current.source, "error", err)
				mu.Lock()
				warnings = append(warnings, fmt.Errorf("failed to fetch versions for %q: %w", current.id, err))
				results[current.id] = 0
				mu.Unlock()
				return
			}

			latest, err := determineLatestTimestamp(r.logger, versions, current.updateType)
			if err != nil {
				r.logger.Warn("Failed to parse version dates for last updated", "asset_id", current.id, "version_count", len(versions), "error", err)
				mu.Lock()
				warnings = append(warnings, fmt.Errorf("failed to resolve latest update for %q: %w", current.id, err))
				results[current.id] = 0
				mu.Unlock()
				return
			}

			mu.Lock()
			results[current.id] = latest
			mu.Unlock()
		}()
	}

	wg.Wait()
	return results, warnings
}

func getLastUpdatedArgs[T any](
	manifests []T,
	idFn func(manifest T) string,
	updateFn func(manifest T) types.UpdateConfig,
) []lastUpdatedArgs {
	sources := make([]lastUpdatedArgs, 0, len(manifests))
	for _, manifest := range manifests {
		id := idFn(manifest)
		update := updateFn(manifest)
		updateType := update.Type
		source := update.Source()
		if id == "" || updateType == "" || source == "" {
			continue
		}

		sources = append(sources, lastUpdatedArgs{
			id:         id,
			updateType: updateType,
			source:     source,
		})
	}
	return sources
}

func determineLatestTimestamp(logger logSink, versions []types.VersionInfo, updateType string) (int64, error) {
	const unset = int64(math.MinInt64)
	bestStable := unset
	bestAny := unset

	for _, version := range versions {
		timestamp, ok := parseVersionDate(version.Date, updateType)
		if !ok {
			continue
		}

		if timestamp > bestAny {
			bestAny = timestamp
		}
		if !version.Prerelease && timestamp > bestStable {
			bestStable = timestamp
		}
	}

	if bestStable != unset {
		return bestStable, nil
	}
	if bestAny != unset {
		logger.Warn("No stable version found, using latest available version", "timestamp", bestAny)
		return bestAny, nil
	}

	return 0, fmt.Errorf("no parseable version dates")
}

func parseVersionDate(value, updateType string) (int64, bool) {
	if value == "" {
		return 0, false
	}

	var layout string
	switch updateType {
	case "github":
		layout = time.RFC3339
	case "custom":
		layout = "2006-01-02"
	default:
		return 0, false
	}

	parsed, err := time.Parse(layout, value)
	if err != nil {
		return 0, false
	}

	return parsed.Unix(), true
}
