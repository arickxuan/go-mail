package util

import (
	"fmt"
	"os"
)

func ListDirectory(path string) (map[string]string, error) {
	entries, err := os.ReadDir(path)
	infos := make(map[string]string, 0)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			//fmt.Printf("[目录] %s\n", entry.Name())
			infos[entry.Name()] = "dir"
		} else {
			info, _ := entry.Info()
			//fmt.Printf("[文件] %s (%d bytes)\n", entry.Name(), info.Size())
			infos[entry.Name()] = fmt.Sprintf("file %d", info.Size())
		}
	}
	return infos, nil
}
