package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	replacementFullPath = map[string]string{
		"@repo1": "repo1-docs/",
		"@repo2": "repo2-docs/",
		"@repo3": "repo3-docs/",
	}
)

func find(root, ext string) []string {
	var files []string
	filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if filepath.Ext(d.Name()) == ext {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func main() {

	for _, s := range find("./", ".md") {
		convertReferences(s)
	}

}

func convertReferences(file string) {

	bytes, err := os.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return
	}

	fileContents := string(bytes) // convert content to a 'string'

	fullLinkRegex := regexp.MustCompile(`\[([^\[]+)\]\((.[^)]*)\)`)
	referenceRegex := regexp.MustCompile(`(@[a-zA-Z0-9]*):.*`)

	res := fullLinkRegex.FindAllStringSubmatch(fileContents, -1)
	for _, item := range res {
		if len(item) != 3 {
			continue // invalid content
		}

		fullLink := item[0]
		linkText := item[1]
		linkUri := item[2]

		found := false
		for keyInDocs, newDirectory := range replacementFullPath {
			newFileContents := replaceReferenceWithCorrectPath(fullLink, linkText, linkUri, keyInDocs, newDirectory, fileContents)
			if newFileContents != "" {
				fileContents = newFileContents
			}
		}
		if !found && referenceRegex.Match([]byte(linkUri)) {
			components := referenceRegex.FindAllStringSubmatch(linkUri, -1)
			if len(components) == 0 {
				fmt.Println("No valid reference to other repositories in link")
			}
			if len(components[0]) < 2 {
				fmt.Println("No valid reference to other repositories in link")
			}
			fmt.Println(" ==> unknown component, no replacement found for", fmt.Sprintf("%s:", components[0][1]), "in", linkUri)
		}
	}

	writeUpdatedContentsBackToFile(file, fileContents)
}

func replaceReferenceWithCorrectPath(fullLink, linkText, linkUri, keyInDocs, newDirectory, originalFileContents string) string {
	if strings.HasPrefix(linkUri, fmt.Sprintf("%s:", keyInDocs)) {
		trimmed := strings.TrimPrefix(linkUri, fmt.Sprintf("%s:", keyInDocs))

		var replacement string
		fileExists := doesReplacementFileExist(newDirectory + trimmed)
		if !fileExists {
			fileExists = doesReplacementFileExist(newDirectory + "docs/" + trimmed)

			if !fileExists {
				fmt.Println(" ==> couldn't find replacement file for trimmed")
				return ""
			}

			mkDocsReference, err := getMkDocsSiteNameForRepo(newDirectory)
			if err != nil {
				fmt.Println("Error during getting mkdocs site name: ", err)
				return ""
			}

			replacement = fmt.Sprintf("[%s](%s/%s)", linkText, mkDocsReference, trimmed)
		} else {

			mkDocsReference, err := getMkDocsSiteNameForRepo(newDirectory)
			if err != nil {
				fmt.Println("Error during getting mkdocs site name: ", err)
				return ""
			}
			replacement = fmt.Sprintf("[%s](%s/%s)", linkText, mkDocsReference, trimmed)
		}

		fmt.Println(" ==> fixing up", fullLink, "with", replacement)
		fileContents := strings.ReplaceAll(originalFileContents, fullLink, replacement)
		return fileContents
	}

	return ""
}

func getMkDocsSiteNameForRepo(folder string) (string, error) {
	var mkDocsReference string
	fileIO, err := os.OpenFile(fmt.Sprintf("%s/mkdocs.yml", folder), os.O_RDWR, 0600)
	if err != nil {
		return "", err
	}
	defer fileIO.Close()
	rawBytes, err := io.ReadAll(fileIO)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(rawBytes), "\n")
	for _, line := range lines {
		if strings.Contains(line, "site_name") {
			mkDocsReference = strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(line, "site_name: ")), " ", "-")
		}
	}

	return mkDocsReference, nil
}

func doesReplacementFileExist(file string) bool {
	_, err := os.OpenFile(file, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return false
	}

	return true
}

func writeUpdatedContentsBackToFile(file, fileContents string) {
	f, err := os.OpenFile(file, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Println("Error during opening file to write contents back: ", err)
		return
	}

	defer f.Close()

	_, err = f.WriteString(fileContents)
	if err != nil {
		fmt.Println("Error during writing contents back to file: ", err)
		return
	}

	// Issue a `Sync` to flush writes to stable storage.
	f.Sync()
}
