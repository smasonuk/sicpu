package utils

import "path/filepath"

func GetPathInfo(relPath string) (fullPath string, parentDir string, err error) {
	// Convert to absolute path (resolves ../../ and cleans the path)
	fullPath, err = filepath.Abs(relPath)
	if err != nil {
		return "", "", err
	}

	// Get the directory containing the file
	parentDir = filepath.Dir(fullPath)

	return fullPath, parentDir, nil
}
