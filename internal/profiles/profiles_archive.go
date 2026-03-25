package profiles

import (
	"archive/tar"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"railyard/internal/files"
	"railyard/internal/paths"
	"railyard/internal/types"
	"railyard/internal/utils"
)

const profileSubscriptionsArchiveFileName = "profile_subscriptions.json"

// CreateProfileArchive generates a tar archive of the profile's current state, including installed maps/mods and their data, and saves it to disk. Returns a GenericResponse indicating success or failure with an appropriate message.
func (s *UserProfiles) CreateProfileArchive(profileID string) types.GenericResponse {
	profile, _, profileErr := s.profileSnapshot(profileID)
	if profileErr != nil {
		s.Logger.Error("Profile not found for archive creation", profileErr, "profile_id", profileID)
		return types.ErrorResponse(profileErr.Error())
	}

	if err := os.MkdirAll(paths.ProfileArchivesPath(), os.ModePerm); err != nil {
		resp, _ := s.archiveError("Failed to create profile archives directory", "failed to create profile archives directory", err, "path", paths.ProfileArchivesPath())
		return resp
	}

	archivePath := profileArchivePath(profile.UUID)

	file, err := os.Create(archivePath)
	if err != nil {
		resp, _ := s.archiveError("Failed to create profile archive file", "failed to create profile archive file", err, "profile_id", profileID, "archive_path", archivePath)
		return resp
	}
	defer file.Close()

	archive := tar.NewWriter(file)
	defer archive.Close()

	tempDir, err := os.MkdirTemp(os.TempDir(), "profile-archive-*")
	if err != nil {
		resp, _ := s.archiveError("Failed to create temporary directory for profile archive", "failed to create temporary directory for profile archive", err, "profile_id", profileID)
		return resp
	}
	defer os.RemoveAll(tempDir)

	if setupErr, ok := s.setupArchiveDirectories(tempDir, profileID); !ok {
		return setupErr
	}

	if mapsErr, ok := s.copyMapsToArchive(tempDir, profileID); !ok {
		return mapsErr
	}

	if modsErr, ok := s.copyModsToArchive(tempDir, profileID); !ok {
		return modsErr
	}

	if metadataErr, ok := s.writeInstalledMetadata(tempDir, profileID); !ok {
		return metadataErr
	}
	if subscriptionsErr, ok := s.writeProfileSubscriptionsMetadata(tempDir, profile.Subscriptions, profileID); !ok {
		return subscriptionsErr
	}

	if err := files.AddDirToArchive(archive, tempDir, tempDir); err != nil {
		resp, _ := s.archiveError("Failed to add temporary profile archive directory to archive", "failed to add temporary profile archive directory to archive", err, "profile_id", profileID)
		return resp
	}

	return types.SuccessResponse(fmt.Sprintf("Profile archive created successfully at %s", archivePath))
}

// setupArchiveDirectories creates the directory structure in the temporary archive directory
func (s *UserProfiles) setupArchiveDirectories(tempDir, profileID string) (types.GenericResponse, bool) {
	if err := os.Mkdir(paths.JoinLocalPath(tempDir, "maps"), os.ModePerm); err != nil {
		return s.archiveError("Failed to create maps directory", "failed to create maps directory", err, "profile_id", profileID)
	}
	if err := os.Mkdir(paths.JoinLocalPath(tempDir, "mods"), os.ModePerm); err != nil {
		return s.archiveError("Failed to create mods directory", "failed to create mods directory", err, "profile_id", profileID)
	}
	return types.GenericResponse{}, true
}

// copyMapsToArchive copies installed maps data to the archive directory
func (s *UserProfiles) copyMapsToArchive(tempDir, profileID string) (types.GenericResponse, bool) {
	for _, mapInfo := range s.Registry.GetInstalledMaps() {
		code := mapInfo.MapConfig.Code
		mapDir := paths.JoinLocalPath(tempDir, "maps", code)

		if err := os.MkdirAll(mapDir, os.ModePerm); err != nil {
			return s.archiveError("Failed to create map directory", "failed to create map directory", err, "profile_id", profileID, "map_id", code)
		}

		// Copy map data
		dataPath := paths.JoinLocalPath(s.Config.Cfg.MetroMakerDataPath, "cities", "data", code)
		if err := files.CopyDirFromFS(paths.JoinLocalPath(mapDir, "data"), os.DirFS(dataPath)); err != nil {
			return s.archiveError("Failed to copy map data", "failed to copy map data", err, "profile_id", profileID, "map_id", code)
		}

		// Copy thumbnail if exists
		thumbnailPath := paths.JoinLocalPath(s.Config.Cfg.MetroMakerDataPath, "public", "data", "city-maps", fmt.Sprintf("%s.svg", code))
		if errResp, ok := files.CopyOptionalFile(thumbnailPath, paths.JoinLocalPath(mapDir, "thumbnail.svg"), profileID, code, "thumbnail", s.Logger); !ok {
			return errResp, false
		}

		// Copy tiles if exists
		tilePath := paths.JoinLocalPath(paths.TilesPath(), fmt.Sprintf("%s.pmtiles", code))
		if errResp, ok := files.CopyOptionalFile(tilePath, paths.JoinLocalPath(mapDir, "tiles.pmtiles"), profileID, code, "tiles", s.Logger); !ok {
			return errResp, false
		}
	}
	return types.GenericResponse{}, true
}

// copyModsToArchive copies installed mods data to the archive directory
func (s *UserProfiles) copyModsToArchive(tempDir, profileID string) (types.GenericResponse, bool) {
	for _, modInfo := range s.Registry.GetInstalledMods() {
		modDest := paths.JoinLocalPath(tempDir, "mods", modInfo.ID)

		if err := os.MkdirAll(modDest, os.ModePerm); err != nil {
			return s.archiveError("Failed to create mod directory", "failed to create mod directory", err, "profile_id", profileID, "mod_id", modInfo.ID)
		}

		modSrc := paths.JoinLocalPath(s.Config.Cfg.GetModsFolderPath(), modInfo.ID)
		if err := files.CopyDirFromFS(paths.JoinLocalPath(modDest, "data"), os.DirFS(modSrc)); err != nil {
			return s.archiveError("Failed to copy mod data", "failed to copy mod data", err, "profile_id", profileID, "mod_id", modInfo.ID)
		}
	}
	return types.GenericResponse{}, true
}

// writeInstalledMetadata writes the installed maps and mods JSON to the archive directory
func (s *UserProfiles) writeInstalledMetadata(tempDir, profileID string) (types.GenericResponse, bool) {
	if err := files.WriteArchiveJSON(tempDir, "installed_maps.json", "installed maps", s.Registry.GetInstalledMaps()); err != nil {
		return s.archiveError("Failed to write installed maps file", "failed to write installed maps file", err, "profile_id", profileID)
	}

	if err := files.WriteArchiveJSON(tempDir, "installed_mods.json", "installed mods", s.Registry.GetInstalledMods()); err != nil {
		return s.archiveError("Failed to write installed mods file", "failed to write installed mods file", err, "profile_id", profileID)
	}
	return types.GenericResponse{}, true
}

// writeProfileSubscriptionsMetadata writes the profile's subscriptions JSON to the archive directory for use in future freshness checks
func (s *UserProfiles) writeProfileSubscriptionsMetadata(
	tempDir string,
	subscriptions types.Subscriptions,
	profileID string,
) (types.GenericResponse, bool) {
	if err := files.WriteArchiveJSON(tempDir, profileSubscriptionsArchiveFileName, "profile subscriptions", subscriptions); err != nil {
		return s.archiveError("Failed to write profile subscriptions file", "failed to write profile subscriptions file", err, "profile_id", profileID)
	}
	return types.GenericResponse{}, true
}

func (s *UserProfiles) RestoreProfileArchive(profileID string) types.GenericResponse {
	profile, _, profileErr := s.profileSnapshot(profileID)
	if profileErr != nil {
		s.Logger.Error("Profile not found for archive restoration", profileErr, "profile_id", profileID)
		return types.ErrorResponse(profileErr.Error())
	}

	archivePath := profileArchivePath(profile.UUID)
	exists, archiveStatusErr := profileArchiveExists(profile.UUID)
	if archiveStatusErr != nil {
		resp, _ := s.archiveError("Failed to check profile archive status", "failed to check profile archive status", archiveStatusErr, "profile_id", profileID, "archive_path", archivePath)
		return resp
	}
	if !exists {
		profileErr := userProfilesError(profileID, "", "", types.ErrorProfileNotFound, "", fmt.Sprintf("Archive file not found for profile restoration: %q", profileID))
		s.Logger.Warn("Profile archive not found for restoration", profileErr, "profile_id", profileID)
		return types.WarnResponse(profileErr.Error())
	}

	tempDir, err := os.MkdirTemp(os.TempDir(), "profile-restore-*")
	if err != nil {
		resp, _ := s.archiveError("Failed to create temporary directory for restoration", "failed to create temporary directory for restoration", err, "profile_id", profileID)
		return resp
	}
	defer os.RemoveAll(tempDir)

	// Extract archive
	if extractErr := files.ExtractArchiveToDir(archivePath, tempDir); extractErr != nil {
		resp, _ := s.archiveError("Failed to extract profile archive", "failed to extract profile archive", extractErr, "profile_id", profileID)
		return resp
	}

	// Load installed maps and mods from archive
	if loadErr, ok := s.loadInstalledFromArchive(tempDir, profileID); !ok {
		return loadErr
	}

	// Restore maps
	if mapsErr, ok := s.restoreMapsFromArchive(tempDir, profileID); !ok {
		return mapsErr
	}

	// Restore mods
	if modsErr, ok := s.restoreModsFromArchive(tempDir, profileID); !ok {
		return modsErr
	}

	// Clean up archive after successful restoration
	os.Remove(archivePath)

	return types.SuccessResponse("Profile archive restoration completed successfully")
}

// loadInstalledFromArchive loads and sets installed maps/mods from the archive metadata
func (s *UserProfiles) loadInstalledFromArchive(tempDir, profileID string) (types.GenericResponse, bool) {
	profileInstalledMapsPath := paths.JoinLocalPath(tempDir, "installed_maps.json")
	if err := s.Registry.SetInstalledMapsFromPath(profileInstalledMapsPath); err != nil {
		return s.archiveError("Failed to set installed maps from archive", "failed to set installed maps from archive", err, "profile_id", profileID)
	}

	profileInstalledModsPath := paths.JoinLocalPath(tempDir, "installed_mods.json")
	if err := s.Registry.SetInstalledModsFromPath(profileInstalledModsPath); err != nil {
		return s.archiveError("Failed to set installed mods from archive", "failed to set installed mods from archive", err, "profile_id", profileID)
	}

	if err := s.Registry.WriteInstalledToDisk(); err != nil {
		return s.archiveError("Failed to write installed to disk", "failed to write installed to disk", err, "profile_id", profileID)
	}

	return types.GenericResponse{}, true
}

// restoreMapsFromArchive restores maps data and metadata from the archive
func (s *UserProfiles) restoreMapsFromArchive(tempDir, profileID string) (types.GenericResponse, bool) {
	for _, mapInfo := range s.Registry.GetInstalledMaps() {
		code := mapInfo.MapConfig.Code

		// Create city data directory
		cityDataPath := paths.JoinLocalPath(s.Config.Cfg.MetroMakerDataPath, "cities", "data", code)
		// Copy city data
		archiveMapDataPath := paths.JoinLocalPath(tempDir, "maps", code, "data")
		if err := files.CopyDirFromFS(cityDataPath, os.DirFS(archiveMapDataPath)); err != nil {
			return s.archiveError("Failed to copy city data from archive", "failed to copy city data from archive", err, "profile_id", profileID, "map_id", code)
		}

		// Restore thumbnail if exists
		archiveThumbnailPath := paths.JoinLocalPath(tempDir, "maps", code, "thumbnail.svg")
		destThumbnailPath := paths.JoinLocalPath(s.Config.Cfg.MetroMakerDataPath, "public", "data", "city-maps", fmt.Sprintf("%s.svg", code))
		if errResp, ok := files.CopyOptionalFile(archiveThumbnailPath, destThumbnailPath, profileID, code, "thumbnail", s.Logger); !ok {
			return errResp, false
		}

		// Restore tiles if exists
		archiveTilePath := paths.JoinLocalPath(tempDir, "maps", code, "tiles.pmtiles")
		destTilePath := paths.JoinLocalPath(paths.TilesPath(), fmt.Sprintf("%s.pmtiles", code))
		if errResp, ok := files.CopyOptionalFile(archiveTilePath, destTilePath, profileID, code, "tiles", s.Logger); !ok {
			return errResp, false
		}
	}
	return types.GenericResponse{}, true
}

// restoreModsFromArchive restores mods data from the archive
func (s *UserProfiles) restoreModsFromArchive(tempDir, profileID string) (types.GenericResponse, bool) {
	for _, modInfo := range s.Registry.GetInstalledMods() {
		modDest := paths.JoinLocalPath(s.Config.Cfg.GetModsFolderPath(), modInfo.ID)

		archiveModDataPath := paths.JoinLocalPath(tempDir, "mods", modInfo.ID, "data")
		if err := files.CopyDirFromFS(modDest, os.DirFS(archiveModDataPath)); err != nil {
			return s.archiveError("Failed to copy mod data from archive", "failed to copy mod data from archive", err, "profile_id", profileID, "mod_id", modInfo.ID)
		}
	}
	return types.GenericResponse{}, true
}

func profileArchivePath(profileUUID string) string {
	return paths.JoinLocalPath(paths.ProfileArchivesPath(), fmt.Sprintf("%s.tar", profileUUID))
}

func profileArchiveExists(profileUUID string) (bool, error) {
	_, err := os.Stat(profileArchivePath(profileUUID))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// isProfileArchiveFresh checks if the profile's current subscriptions match the subscriptions stored in the archive, indicating whether the archive is fresh or stale.
func (s *UserProfiles) isProfileArchiveFresh(profile types.UserProfile) (bool, error) {
	archivePath := profileArchivePath(profile.UUID)
	exists, archiveStatusErr := profileArchiveExists(profile.UUID)
	if archiveStatusErr != nil {
		return false, archiveStatusErr
	}
	if !exists {
		return false, nil
	}

	archiveSubscriptions, found, err := files.ReadJSONFromTarArchive[types.Subscriptions](archivePath, profileSubscriptionsArchiveFileName)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}
	return areSubscriptionsEqual(profile.Subscriptions, archiveSubscriptions), nil
}

func areSubscriptionsEqual(left types.Subscriptions, right types.Subscriptions) bool {
	return utils.MapEqual(left.Maps, right.Maps) &&
		utils.MapEqual(left.LocalMaps, right.LocalMaps) &&
		utils.MapEqual(left.Mods, right.Mods)
}
