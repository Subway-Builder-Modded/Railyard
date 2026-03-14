package types

type RailyardVersionInfo struct {
	Version                 string `json:"version"`
	Name                    string `json:"name"`
	Changelog               string `json:"changelog"`
	Date                    string `json:"date"`
	Prerelease              bool   `json:"prerelease"`
	MacOSDownloadURL        string `json:"macos_download_url,omitempty"`
	WindowsARMDownloadURL   string `json:"windows_arm_download_url,omitempty"`
	WindowsX64DownloadURL   string `json:"windows_x64_download_url,omitempty"`
	LinuxCurrentDownloadURL string `json:"linux_download_url,omitempty"`
}
