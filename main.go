package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"sort"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/google/shlex"
)

var nameRegex = regexp.MustCompile("[^-.+a-zA-Z0-9]+")

func main() {
	reeveAPI := os.Getenv("REEVE_API")
	if reeveAPI == "" {
		fmt.Println("This docker image is a Reeve CI pipeline step and is not intended to be used on its own.")
		os.Exit(1)
	}

	filePatterns, err := shlex.Split(os.Getenv("FILES"))
	if err != nil {
		fmt.Printf("Error parsing file pattern list - %s\n", err)
		os.Exit(1)
	}
	files := make([]string, 0, len(filePatterns))
	for _, pattern := range filePatterns {
		matches, err := doublestar.FilepathGlob(pattern, doublestar.WithFilesOnly())
		if err != nil {
			fmt.Printf("Error parsing file pattern \"%s\" - %s\n", pattern, err)
			os.Exit(1)
		}
		files = append(files, matches...)
	}
	files = distinct(files)
	sort.Strings(files)

	apiUrl := os.Getenv("API_URL")
	if apiUrl == "" {
		fmt.Println("Missing API url")
		os.Exit(1)
	}

	apiUser := os.Getenv("API_USER")
	if apiUser == "" {
		fmt.Println("Missing API user")
		os.Exit(1)
	}

	apiPassword := os.Getenv("API_PASSWORD")
	if apiPassword == "" {
		fmt.Println("Missing API password")
		os.Exit(1)
	}

	packageOwner := os.Getenv("PACKAGE_OWNER")
	if packageOwner == "" {
		fmt.Println("Missing package owner")
		os.Exit(1)
	}

	packageName := os.Getenv("PACKAGE_NAME")
	if packageName == "" {
		fmt.Println("Missing package name")
		os.Exit(1)
	}

	packageVersion := os.Getenv("PACKAGE_VERSION")
	if packageVersion == "" {
		fmt.Println("Missing package version")
		os.Exit(1)
	}

	skipExisting := os.Getenv("SKIP_EXISTING") == "true"

	var packageFiles map[string]struct{}
	if skipExisting {
		requestUrl := fmt.Sprintf("%s/api/v1/packages/%s/generic/%s/%s/files", apiUrl, url.PathEscape(packageOwner), url.PathEscape(packageName), url.PathEscape(packageVersion))
		request, err := http.NewRequest(http.MethodGet, requestUrl, nil)
		if err != nil {
			fmt.Printf("Error fetching package file list - %s\n", err)
			os.Exit(1)
		}
		request.SetBasicAuth(apiUser, apiPassword)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			fmt.Printf("Error fetching package file list - %s\n", err)
			os.Exit(1)
		}
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			if response.StatusCode != http.StatusNotFound {
				fmt.Printf("Fetching package file list returned status %v\n", response.StatusCode)
				os.Exit(1)
			}
		} else {
			var packageList []struct {
				Name string `json:"name"`
			}
			err := json.NewDecoder(response.Body).Decode(&packageList)
			response.Body.Close()
			if err != nil {
				fmt.Printf("Error fetching package file list - %s\n", err)
				os.Exit(1)
			}
			packageFiles = make(map[string]struct{}, len(packageList))
			for _, packageFile := range packageList {
				packageFiles[packageFile.Name] = struct{}{}
			}
		}
	}

	for _, filename := range files {
		targetName := nameRegex.ReplaceAllString(path.Base(filename), "_")

		if _, ok := packageFiles[targetName]; ok {
			fmt.Printf("Skipping \"%s\" because %s already exists in the package\n", filename, targetName)
			continue
		}

		fmt.Printf("Uploading \"%s\" as %s...\n", filename, targetName)

		file, err := os.Open(filename)
		if err != nil {
			fmt.Printf("Error uploading file \"%s\" - %s\n", filename, err)
			os.Exit(1)
		}
		requestUrl := fmt.Sprintf("%s/api/packages/%s/generic/%s/%s/%s", apiUrl, url.PathEscape(packageOwner), url.PathEscape(packageName), url.PathEscape(packageVersion), url.PathEscape(targetName))
		request, err := http.NewRequest(http.MethodPut, requestUrl, file)
		if err != nil {
			fmt.Printf("Error uploading file \"%s\" - %s\n", filename, err)
			os.Exit(1)
		}
		request.SetBasicAuth(apiUser, apiPassword)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			fmt.Printf("Error uploading file \"%s\" - %s\n", filename, err)
			os.Exit(1)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusCreated {
			fmt.Printf("Uploading file returned status %v\n", response.StatusCode)
			os.Exit(1)
		}
	}

	// DOESN'T WORK WITH CURRENT FORGEJO (POST /.../link fails if already linked, POST /.../unlink fails always)
	/*
		if packageRepository := os.Getenv("PACKAGE_REPOSITORY"); packageRepository != "" {
			fmt.Printf("Linking package \"%s\" to repository \"%s\"...\n", packageName, packageRepository)

			requestUrl := fmt.Sprintf("%s/api/v1/packages/%s/generic/%s/-/link/%s", apiUrl, url.PathEscape(packageOwner), url.PathEscape(packageName), url.PathEscape(packageRepository))
			request, err := http.NewRequest(http.MethodPost, requestUrl, nil)
			if err != nil {
				fmt.Printf("Error linking repository - %s\n", err)
				os.Exit(1)
			}
			request.SetBasicAuth(apiUser, apiPassword)
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				fmt.Printf("Error linking repository - %s\n", err)
				os.Exit(1)
			}
			response.Body.Close()
			if response.StatusCode != http.StatusCreated {
				fmt.Printf("Linking repository returned status %v\n", response.StatusCode)
				os.Exit(1)
			}
		}
	*/

	fmt.Println("Done")
}

func distinct[T comparable](items []T) []T {
	keys := make(map[T]struct{})
	result := make([]T, 0, len(items))
	for _, item := range items {
		if _, exists := keys[item]; !exists {
			keys[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}
