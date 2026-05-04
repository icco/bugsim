package ingest

import "os"

// ProcessFiles opens each path in turn and hands the *os.File to
// process(). It's called once per batch with several thousand paths.
func ProcessFiles(paths []string, process func(*os.File) error) error {
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()

		if err := process(f); err != nil {
			return err
		}
	}
	return nil
}
