package main

import (
	"fmt"
	"os"
	"path"
)

const PLAT = "linux"
const ARCH = "amd64"

func (a *App) Launch() {
	apppath := path.Join(a.Path, "core", "waveviewer")
	env := os.Environ()
	env = append(env, "LD_LIBRARY_PATH="+path.Join(a.Path, "core"), "QT_PLUGIN_PATH="+a.Path, "QML2_IMPORT_PATH="+path.Join(a.Path, "qml"))
	err := os.Chmod(apppath, 0777)
	if err != nil {
		fmt.Printf("[ERR] Could not make app executable: %s\n", err)
		os.Exit(1)
	}
	p, err := os.StartProcess(apppath, os.Args, &os.ProcAttr{Env: env, Files: []*os.File{nil, os.Stdout, os.Stderr}})
	if err != nil {
		fmt.Printf("[ERR] Could not launch app: %s\n", err)
		os.Exit(1)
	}
	ps, _ := p.Wait()
	os.Stdout.Sync()
	os.Stderr.Sync()
	fmt.Println(" == CHILD EXIT :", ps.String(), "==")
}
