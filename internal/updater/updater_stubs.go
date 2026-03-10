//go:build !windows

package updater

func launchElevated(exePath string, args string, workingDir string) error {
	return nil
}
