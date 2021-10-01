package utils

import (
	"io/ioutil"
	"log"
	"os"

	yaml "gopkg.in/yaml.v2"
)

// DeployConfig contains all info about deployment in k8s
type DeployConfig struct {
	Certs      CertsConf                 `yaml:"certs"`
	ServerPort int                       `yaml:"server-port"`
	Providers  map[string]ProviderConfig `yaml:"providers"`
}

// CertsConf contains location of key/cert files
type CertsConf struct {
	KeyFile  string `yaml:"key-file"`
	CertFile string `yaml:"cert-file"`
}

// ProviderConfig contains info about git provider
type ProviderConfig struct {
	URL    string `yaml:"url"`          // base url of git provider
	Type   string `yaml:"api-type"`     // may be gitlab or github
	Secret string `yaml:"secret-token"` // api access secret token
}

// ProjectConfig contains info about git provider
type ProjectConfig struct {
	Provider          string `yaml:"provider"`       // provider of git
	Group             string `yaml:"group"`          // group or oraganization
	Project           string `yaml:"project"`        // project or repo
	SourcePath        string `yaml:"source-path"`    // path in zip file
	MountVolume       string `yaml:"mount-volume"`   // volume where copy release artifacts
	AppServerMode     string `yaml:"app-serve-mode"` // `master` (prod) or `devel` mode
	AssetsSource      string `yaml:"assets-src"`     // source of additional assets
	AssetsDestination string `yaml:"assets-dst"`     // name of folder in mount volume for copy assets
}

// LoadDeployConfig load config of deployment
func LoadDeployConfig(path string) *DeployConfig {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("%s get err #%v", path, err)
	}
	var config = DeployConfig{}

	if err = yaml.Unmarshal(file, &config); err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return &config
}

// LoadProjectConfig load project description from environment variables
func LoadProjectConfig() *ProjectConfig {
	provider := os.Getenv("PROVIDER")
	group := os.Getenv("GROUP")
	project := os.Getenv("PROJECT")
	sourcePath := os.Getenv("SOURCE_PATH")
	volume := os.Getenv("VOLUME")
	mode := os.Getenv("APP_SERVER_MODE")
	assetsSource := os.Getenv("ASSETS_SRC")
	assetsDestination := os.Getenv("ASSETS_DST")
	return &ProjectConfig{provider, group, project, sourcePath, volume, mode, assetsSource, assetsDestination}
}
