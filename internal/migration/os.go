package migration

import "os"

func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
