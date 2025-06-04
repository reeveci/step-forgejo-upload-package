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
		panic(fmt.Sprintf("error parsing file pattern list - %s", err))
	}
	files := make([]string, 0, len(filePatterns))
	for _, pattern := range filePatterns {
		matches, err := doublestar.FilepathGlob(pattern, doublestar.WithFilesOnly())
		if err != nil {
			panic(fmt.Sprintf(`error parsing file pattern "%s" - %s`, pattern, err))
		}
		files = append(files, matches...)
	}
	files = distinct(files)
	sort.Strings(files)

	apiUrl := os.Getenv("API_URL")
	if apiUrl == "" {
		panic("missing API url")
	}

	apiUser := os.Getenv("API_USER")
	if apiUser == "" {
		panic("missing API user")
	}

	apiPassword := os.Getenv("API_PASSWORD")
	if apiPassword == "" {
		panic("missing API password")
	}

	packageOwner := os.Getenv("PACKAGE_OWNER")
	if packageOwner == "" {
		panic("missing package owner")
	}

	packageName := os.Getenv("PACKAGE_NAME")
	if packageName == "" {
		panic("missing package name")
	}

	packageVersion := os.Getenv("PACKAGE_VERSION")
	if packageVersion == "" {
		panic("missing package version")
	}

	packageRepository := os.Getenv("PACKAGE_REPOSITORY")

	var packageFiles map[string]struct{}
	requestUrl := fmt.Sprintf("%s/api/v1/packages/%s/generic/%s/%s/files", apiUrl, url.PathEscape(packageOwner), url.PathEscape(packageName), url.PathEscape(packageVersion))
	request, err := http.NewRequest(http.MethodGet, requestUrl, nil)
	if err != nil {
		panic(fmt.Sprintf(`error fetching package file list - %s`, err))
	}
	request.SetBasicAuth(apiUser, apiPassword)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(fmt.Sprintf(`error fetching package file list - %s`, err))
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			panic(fmt.Sprintf("fetching package file list returned status %v", response.StatusCode))
		}
	} else {
		var packageList []struct {
			Name string `json:"name"`
		}
		err := json.NewDecoder(response.Body).Decode(&packageList)
		response.Body.Close()
		if err != nil {
			panic(fmt.Sprintf(`error fetching package file list - %s`, err))
		}
		packageFiles = make(map[string]struct{}, len(packageList))
		for _, packageFile := range packageList {
			packageFiles[packageFile.Name] = struct{}{}
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
			panic(fmt.Sprintf(`error uploading file "%s" - %s`, filename, err))
		}
		requestUrl := fmt.Sprintf("%s/api/packages/%s/generic/%s/%s/%s", apiUrl, url.PathEscape(packageOwner), url.PathEscape(packageName), url.PathEscape(packageVersion), url.PathEscape(targetName))
		request, err := http.NewRequest(http.MethodPut, requestUrl, file)
		if err != nil {
			panic(fmt.Sprintf(`error uploading file "%s" - %s`, filename, err))
		}
		request.SetBasicAuth(apiUser, apiPassword)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			panic(fmt.Sprintf(`error uploading file "%s" - %s`, filename, err))
		}
		response.Body.Close()
		if response.StatusCode != http.StatusCreated {
			panic(fmt.Sprintf("uploading file returned status %v", response.StatusCode))
		}
	}

	if packageRepository != "" {
		fmt.Printf("Linking package \"%s\" to repository \"%s\"...\n", packageName, packageRepository)

		requestUrl := fmt.Sprintf("%s/api/v1/packages/%s/generic/%s/-/link/%s", apiUrl, url.PathEscape(packageOwner), url.PathEscape(packageName), url.PathEscape(packageRepository))
		request, err := http.NewRequest(http.MethodPost, requestUrl, nil)
		if err != nil {
			panic(fmt.Sprintf(`error linking repository - %s`, err))
		}
		request.SetBasicAuth(apiUser, apiPassword)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			panic(fmt.Sprintf(`error linking repository - %s`, err))
		}
		response.Body.Close()
		if response.StatusCode != http.StatusCreated {
			panic(fmt.Sprintf("linking repository returned status %v", response.StatusCode))
		}
	}

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
