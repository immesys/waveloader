package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/kardianos/osext"
	"gopkg.in/yaml.v2"
)

const MANIFEST = "http://get.bw2.io/wavelet/1.x/manifest.yaml"
const TIMEOUT = 5 * time.Second

type Manifest struct {
	MF map[string]map[string][]string
}
type App struct {
	Path  string
	Major int
	Minor int
	Build int
}

func GetManifest() chan *Manifest {
	rv := make(chan *Manifest, 1)
	go func() {
		resp, err := http.Get(MANIFEST)
		if err != nil {
			fmt.Printf("[ERR] Could not check for new version: %s\n", err)
			close(rv)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("[ERR] Problem reading manifest: %s\n", err)
			close(rv)
			return
		}
		var mf map[string]map[string][]string
		err = yaml.Unmarshal(body, &mf)
		if err != nil {
			fmt.Printf("[ERR] Problem with manifest: %s\n", err)
			close(rv)
			return
		}
		rv <- &Manifest{MF: mf}
	}()
	return rv
}
func TryGetManifest() *Manifest {
	rc := GetManifest()
	select {
	case rv, ok := <-rc:
		if !ok {
			return nil
		}
		return rv
	case <-time.After(TIMEOUT):
		return nil
	}
}
func (*Manifest) FindLocal() []*App {
	us, err := osext.Executable()
	if err != nil {
		fmt.Println("[ABORT] could not locate launcher:", err)
		os.Exit(1)
	}
  contents, err := ioutil.ReadDir(path.Join(path.Dir(us),"versions")
  if err != nil {
    fmt.Println("[ABORT] could not check current versions:", err)
    os.Exit(1)
  }
  rv := []*App{}
  for _, d := range contents {
    if !d.IsDir() {
      continue
    }
    toks := strings.Split(d.Name(),".")
    if len(toks) != 3 {
      fmt.Println("[ABORT] bad version: ",d.Name())
      os.Exit(1)
    }
    maj, err := strconv.ParseInt(toks[0],10,64)
    if err != nil {
      fmt.Println("[ABORT] bad version: ",d.Name())
      os.Exit(1)
    }
    min, err := strconv.ParseInt(toks[1],10,64)
    if err != nil {
      fmt.Println("[ABORT] bad version: ",d.Name())
      os.Exit(1)
    }
    bld, err := strconv.ParseInt(toks[2],10,64)
    if err != nil {
      fmt.Println("[ABORT] bad version: ",d.Name())
      os.Exit(1)
    }
    a := App {
      Major: maj,
      Minor: min,
      Build: bld,
      Path:d
    }
  }
}
func main() {
	mf := TryGetManifest()
	if mf == nil {
		app := mf.FindNewestLocal()
		if app != nil {
			app.Launch()
		} else {
			fmt.Println("Could not download latest version, and no local version found")
		}
	}
}
