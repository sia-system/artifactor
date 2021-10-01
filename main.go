package main

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gookit/color"

	githandler "demius.md/artifactor/git-handler"
	utils "demius.md/artifactor/utils"
)

func main() {
	executablePath := utils.ExecutableDir()
	homePath := utils.UserHomeDir()
	fmt.Println("executable path: " + executablePath)
	fmt.Println("      home path: " + homePath)

	// disable check of http cert because wrong acc cert and Marina gitlab
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	gitClients := make(map[string]githandler.GitClient)

	config := utils.LoadDeployConfig("/etc/artifactor/config.yaml")
	projectConf := utils.LoadProjectConfig()

	for provider, providerConf := range config.Providers {
		color.FgGray.Println(" provider " + provider)
		var gitclient githandler.GitClient

		if val, ok := gitClients[provider]; ok {
			gitclient = val
		} else {
			if providerConf.Type == "gitlab" {
				gitclient = githandler.ConnectGitlab(provider, providerConf.Secret)
			} else if providerConf.Type == "github" {
				gitclient = githandler.ConnectGithub(provider, providerConf.Secret)
			} else {
				color.FgRed.Println("Unknwn provider type: " + providerConf.Type)
				continue
			}

			gitClients[provider] = gitclient
		}
	}

	{
		color.FgGray.Println(" project: " + projectConf.Project)
		color.FgGray.Println(" mount-volume: " + projectConf.MountVolume)
		color.FgGray.Println(" source-path: " + projectConf.Path)

		gitclient, ok := gitClients[projectConf.Provider]
		if !ok {
			color.FgRed.Println("Not found provider: " + projectConf.Provider)
			return
		}

		assets, err := gitclient.LoadAssets(projectConf.Group, projectConf.Project, projectConf.AppServerMode)
		if err != nil {
			color.FgRed.Printf("%v: \n", err)
		} else if assets != nil {
			if err := unzipAssets(projectConf.MountVolume, projectConf.Path, assets); err != nil {
				color.FgRed.Printf("%v\n", err)
			}
			if err := copyAssets(projectConf.AssetsSource, projectConf.AssetsDestination, projectConf.MountVolume); err != nil {
				color.FgRed.Printf("%v\n", err)
			}
		} else {
			color.FgGray.Println("     not found assets")
		}
	}

	println()
	color.FgLightCyan.Println("OK")
}

// unzip assets from git server to destination folder
func unzipAssets(destination, subpath string, body []byte) error {
	buff := bytes.NewBuffer(body)
	reader := bytes.NewReader(buff.Bytes())

	// Open a zip archive for reading.
	zipReader, err := zip.NewReader(reader, int64(len(body)))
	if err != nil {
		return fmt.Errorf("unzip assets error: %v", err)
	}
	println()

	cnt := 0
	for _, f := range zipReader.File {
		filename := f.Name

		if filename == "" {
			continue
		}

		fmt.Println("        found entry: " + filename)

		if len(subpath) > 0 {
			if !strings.HasPrefix(filename, subpath) {
				continue
			}
			filename = filename[len(subpath)+1:]
			if filename == "" {
				continue
			}
		}

		fpath := filepath.Join(destination, filename)

		fmt.Println("        uzip entry: " + fpath)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(destination)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal file path", fpath)
		}

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
		cnt++
	}
	color.FgGray.Printf("     unzipped to `%s`  %v files\n", destination, cnt)

	return nil
}

// Separator is file separator for current OS
const Separator = string(filepath.Separator)

// copy additional assets from mounted source dir to destination folder
// currently supports only flat directories
func copyAssets(assetsSource, assetsDestination, copyLocation string) error {
	if assetsSource == "" {
		return nil
	}
	color.FgGray.Printf("     copy additional assets\n")

	var sourcePrefixLen = len(assetsSource)

	var destinationDir = filepath.Join(copyLocation, assetsDestination)
	os.MkdirAll(destinationDir, os.ModePerm)

	return filepath.Walk(assetsSource, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error walk dir%s: %v", path, err)
		}
		var strippedFilename = path[sourcePrefixLen:]
		println("src path: " + path + "; stripped: " + strippedFilename)
		var filename = filepath.Join(destinationDir, strippedFilename)
		println("destination: " + filename)

		if f.IsDir() {
			if err = createDir(filename); err != nil {
				return err
			}
			return nil
		}

		source, err := os.Open(path)
		if err != nil {
			return err
		}
		defer source.Close()
		// println("source file successfully opened")

		outFile, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer outFile.Close()
		// println("destination file successfully opened")

		_, err = io.Copy(outFile, source)

		return err
	})
}

func createDir(destination string) error {
	if len(destination) > 0 && destination != "./" {
		_, err := os.Stat(destination)
		if err != nil {
			if os.IsNotExist(err) {
				println("create directory: " + destination)
				if err = os.MkdirAll(destination, os.ModePerm); err != nil {
					return fmt.Errorf("error creation dir %s: %v", destination, err)
				}
			}
		}
	}
	return nil
}
