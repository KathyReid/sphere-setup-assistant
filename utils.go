package main

import (
	"io"
	"os"
)

func WriteToFile(filename string, contents string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = io.WriteString(f, contents)
	if err != nil {
		return err
	}

	return nil
}
