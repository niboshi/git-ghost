package git

import (
	"bytes"
	"errors"
	"fmt"
	"git-ghost/pkg/util"
	"io"
	"os"
	"os/exec"

	log "github.com/Sirupsen/logrus"
	multierror "github.com/hashicorp/go-multierror"
)

func CreateDiffBundleFile(dir, filepath, fromComittish, toComittish string) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	cmd := exec.Command("git", "-C", dir,
		"log", "-p", "--reverse", "--pretty=email", "--stat", "-m", "--first-parent", "--binary",
		fmt.Sprintf("%s..%s", fromComittish, toComittish),
	)
	stderr := bytes.NewBufferString("")
	cmd.Stderr = stderr
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer reader.Close()
	err = cmd.Start()
	if err != nil {
		s := stderr.String()
		if s != "" {
			return errors.New(s)
		}
		return err
	}

	total := 0
	buf := make([]byte, 1024, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			d := buf[:n]
			_, err = f.Write(d)
			if err != nil {
				return err
			}
			total += n
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return nil
}

func ApplyDiffBundleFile(dir, filepath string) error {
	var errs error
	err := util.JustRunCmd(
		exec.Command("git", "-C", dir, "am", filepath),
	)
	if err != nil {
		multierror.Append(errs, err)
		log.WithFields(util.MergeFields(
			log.Fields{
				"srcDir":   dir,
				"filepath": filepath,
				"error":    err.Error(),
			})).Info("apply('git am') failed. aborting.")
		resetErr := util.JustRunCmd(
			exec.Command("git", "-C", dir, "am", "--abort"),
		)
		if resetErr != nil {
			multierror.Append(errs, resetErr)
		}
	}
	return errs
}

func CreateDiffPatchFile(dir, filepath, comittish string) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	cmd := exec.Command("git", "-C", dir, "diff", "--patience", "--binary", comittish)
	stderr := bytes.NewBufferString("")
	cmd.Stderr = stderr
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer reader.Close()
	err = cmd.Start()
	if err != nil {
		s := stderr.String()
		if s != "" {
			return errors.New(s)
		}
		return err
	}

	total := 0
	buf := make([]byte, 1024, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			d := buf[:n]
			_, err = f.Write(d)
			if err != nil {
				return err
			}
			total += n
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return nil
}

func ApplyDiffPatchFile(dir, filepath string) error {
	return util.JustRunCmd(
		exec.Command("git", "-C", dir, "apply", filepath),
	)
}
