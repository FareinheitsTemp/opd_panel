package cmd

import "os/exec"

func execStart(name string, args ...string) error {
	return exec.Command(name, args...).Start()
}
