package libcore

import (
	"io"
	"os"

	"github.com/ulikunitz/xz"
)

func Unxz(archive string, path string) error {
	i, err := os.Open(archive)
	if err != nil {
		return err
	}
	r, err := xz.NewReader(i)
	if err != nil {
		closeIgnore(i)
		return err
	}
	o, err := os.Create(path)
	if err != nil {
		closeIgnore(i)
		return err
	}
	_, err = io.Copy(o, r)
	closeIgnore(i, o)
	return err
}

func unxz(path string) error {
	err := Unxz(path, path+".tmp")
	if err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}
