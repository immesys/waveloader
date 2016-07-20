package main

import (
	"fmt"
	"os"
	"path"
)

const PLAT = "darwin"
const ARCH = "amd64"

func (a *App) Launch() {
	apppath := path.Join(a.Path, "waveviewer.app", "Contents", "MacOS", "waveviewer")
	env := os.Environ()
	env = append(env, "QT_PLUGIN_PATH="+path.Join(a.Path, "Contents", "plugins"), "QML2_IMPORT_PATH="+path.Join(a.Path, "Contents", "qml"))
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
