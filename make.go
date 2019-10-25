package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var username = os.Getenv("USERNAME")

func main() {
	force := flag.Bool("f", false, "ignore system check")
	flag.Parse()

	if runtime.GOOS == "windows" && !*force {
		_, err := exec.LookPath("pacman")
		if err != nil {
			fmt.Fprintln(os.Stderr, "pacman not found; not in MSYS?")
			os.Exit(1)
		}
	}

	qmk_init()

	build("preonic", "rev3")
}

// build builds the keyboard version for the current user
func build(keyboard, version string) {
	p := filepath.Join("qmk/keyboards", keyboard, "keymaps", username)
	syncdir(keyboard, p)

	with_chdir("qmk", func() {
		run("make", fmt.Sprint(keyboard, "/", version, ":", username))
	})

	verify(os.MkdirAll("build", 0777))

	matches, err := filepath.Glob(filepath.Join("qmk/.build",
		fmt.Sprint(keyboard, "_", version, "_", username, ".*")))
	verify(err)

	for _, m := range matches {
		b := filepath.Base(m)
		verify(os.Rename(m, filepath.Join("build", b)))
	}

	verify(os.RemoveAll(p))
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func verify(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func qmk_init() {
	if !exists("qmk/Makefile") {
		run("git", "submodule", "update", "--init", "--recursive")
		with_chdir("qmk", func() {
			run("make", "git-submodule")
		})
	}
}

func with_chdir(dir string, fn func()) {
	wd, err := os.Getwd()
	verify(err)

	verify(os.Chdir(dir))
	defer os.Chdir(wd)

	fn()
}

func run(cmd string, args ...string) {
	x := exec.Command(cmd, args...)
	x.Stdin = os.Stdin
	x.Stdout = os.Stdout
	x.Stderr = os.Stderr
	verify(x.Run())
}

// syncdir ensures src is accessible through link
func syncdir(src, link string) error {
	if runtime.GOOS != "windows" {
		if exists(link) {
			return nil
		}
		rsrc, err := filepath.Rel(filepath.Dir(link), src)
		if err != nil {
			return err
		}
		return os.Symlink(rsrc, link)
	}

	// symlink doesn't seem to work with QMK Firmware 0.7.29

	// /mir: mirror mode: copy recursively and
	//       delete files and directories missing in src from destination
	// /njh: no job header
	// /njs: no job summary
	x := exec.Command("robocopy", "/mir", "/njh", "/njs",
		filepath.Clean(src), filepath.Clean(link))

	x.Stdout = os.Stdout
	x.Stderr = os.Stderr

	return x.Run()
}
