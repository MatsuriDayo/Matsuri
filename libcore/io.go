package libcore

import (
	"io"
	"libcore/comm"
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
		comm.CloseIgnore(i)
		return err
	}
	o, err := os.Create(path)
	if err != nil {
		comm.CloseIgnore(i)
		return err
	}
	_, err = io.Copy(o, r)
	comm.CloseIgnore(i, o)
	return err
}
