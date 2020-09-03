package githandler

// GitClient is abstraction over Github and Gitlab
type GitClient interface {
	LoadAssets(groupName, projectName, mode string) ([]byte, error)
	ProviderName() string
}
