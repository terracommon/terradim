package model

import (
	"io"
	"os"
)

// Copy file or dir from src to dst
func Copy(src, dst string) (err error) {
	var info os.FileInfo

	if info, err = os.Stat(src); err != nil {
		return
	}
	if err = os.RemoveAll(dst); err != nil {
		return
	}

	if info.IsDir() {
		err = copyDir(src, dst, info)
		return
	}
	err = copyFile(src, dst, info)
	return
}

func copyDir(src, dst string, srcInfo os.FileInfo) (err error) {
	err = os.MkdirAll(dst, srcInfo.Mode())
	return
}

func copyFile(src, dst string, srcInfo os.FileInfo) (err error) {
	var srcFd, dstFd *os.File

	if srcFd, err = os.Open(src); err != nil {
		return
	}
	defer srcFd.Close()

	if dstFd, err = os.Create(dst); err != nil {
		return
	}
	defer dstFd.Close()

	if _, err = io.Copy(dstFd, srcFd); err != nil {
		return
	}
	err = os.Chmod(dst, srcInfo.Mode())
	return
}
