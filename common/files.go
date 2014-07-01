package common

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

type TemporaryFile interface {
	Path() string
	Release()
}

type TemporaryFileManager interface {
	Notify(path string)
	Create(path string) TemporaryFile
	List() map[string]int
}

type defaultTemporaryFile struct {
	tfm  TemporaryFileManager
	path string
}

type defaultTemporaryFileManager struct {
	files map[string]int
	mu    sync.Mutex
}

func NewTemporaryFileManager() TemporaryFileManager {
	tfm := new(defaultTemporaryFileManager)
	tfm.files = make(map[string]int)
	return tfm
}

func (tf *defaultTemporaryFile) Path() string {
	return tf.path
}

func (tf *defaultTemporaryFile) Release() {
	go func() {
		time.Sleep(1 * time.Minute)
		tf.tfm.Notify(tf.path)
	}()
}

func (tfm *defaultTemporaryFileManager) Notify(path string) {
	tfm.mu.Lock()
	defer tfm.mu.Unlock()
	count, hasCount := tfm.files[path]
	if hasCount {
		count = count - 1
		if count > 0 {
			tfm.files[path] = count
			return
		}
		delete(tfm.files, path)
		err := os.Remove(path)
		if err != nil {
			log.Println(err)
		}
	}
}

func (tfm *defaultTemporaryFileManager) Create(path string) TemporaryFile {
	tfm.mu.Lock()
	defer tfm.mu.Unlock()
	count, hasCount := tfm.files[path]
	if hasCount {
		tfm.files[path] = count + 1
		return &defaultTemporaryFile{tfm, path}
	}
	tfm.files[path] = 1
	return &defaultTemporaryFile{tfm, path}
}

func (tfm *defaultTemporaryFileManager) List() map[string]int {
	tfm.mu.Lock()
	defer tfm.mu.Unlock()
	results := make(map[string]int)
	for path, count := range tfm.files {
		results[path] = count
	}
	return results
}

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = CopyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func CopyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
