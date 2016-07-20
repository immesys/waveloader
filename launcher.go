package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"
)

//This loader is designed purely for version 1
const REPO = "http://get.bw2.io/wavelet/1.x/" + PLAT + "/" + ARCH + "/"
const MANIFEST = REPO + "manifest.yaml"
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

func datadir() string {
	hd, err := homedir.Dir()
	if err != nil {
		fmt.Println("Could not locate home directory:", err)
		os.Exit(1)
	}
	dd := path.Join(hd, ".waveloader")
	return dd
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
	basedir := path.Join(datadir(), "versions")
	contents, err := ioutil.ReadDir(basedir)
	if err != nil {
		fmt.Println("[ABORT] could not check current versions:", err)
		os.Exit(1)
	}
	rv := []*App{}
	for _, d := range contents {
		if !d.IsDir() {
			continue
		}
		toks := strings.Split(d.Name(), ".")
		if len(toks) != 3 {
			fmt.Println("[ABORT] bad version: ", d.Name())
			os.Exit(1)
		}
		maj, err := strconv.ParseInt(toks[0], 10, 64)
		if err != nil {
			fmt.Println("[ABORT] bad version: ", d.Name())
			os.Exit(1)
		}
		min, err := strconv.ParseInt(toks[1], 10, 64)
		if err != nil {
			fmt.Println("[ABORT] bad version: ", d.Name())
			os.Exit(1)
		}
		bld, err := strconv.ParseInt(toks[2], 10, 64)
		if err != nil {
			fmt.Println("[ABORT] bad version: ", d.Name())
			os.Exit(1)
		}
		a := App{
			Major: int(maj),
			Minor: int(min),
			Build: int(bld),
			Path:  path.Join(basedir, d.Name()),
		}
		rv = append(rv, &a)
	}
	return rv
}
func (m *Manifest) Latest() (int, int, int) {
	mmaj := int64(0)
	mmin := int64(0)
	mbld := int64(0)
	for ver, _ := range m.MF {
		toks := strings.Split(ver, ".")
		if len(toks) != 3 {
			fmt.Println("[ABORT] bad mf version: ", ver)
			os.Exit(1)
		}
		maj, err := strconv.ParseInt(toks[0], 10, 64)
		if err != nil {
			fmt.Println("[ABORT] bad mf version: ", ver)
			os.Exit(1)
		}
		min, err := strconv.ParseInt(toks[1], 10, 64)
		if err != nil {
			fmt.Println("[ABORT] bad mf version: ", ver)
			os.Exit(1)
		}
		bld, err := strconv.ParseInt(toks[2], 10, 64)
		if err != nil {
			fmt.Println("[ABORT] bad mf version: ", ver)
			os.Exit(1)
		}
		if maj > mmaj {
			mmaj = maj
			mmin = min
			mbld = bld
		} else if maj == mmaj {
			if min > mmin {
				mmaj = maj
				mmin = min
				mbld = bld
			} else if min == mmin {
				if bld > mbld {
					mmaj = maj
					mmin = min
					mbld = bld
				}
			}
		}
	}
	return int(mmaj), int(mmin), int(mbld)
}
func (m *Manifest) Download(maj, min, bld int) {
	ver := fmt.Sprintf("%d.%d.%d", maj, min, bld)
	fmt.Println("Checking integrity of", ver)
	entry, ok := m.MF[ver]
	if !ok {
		fmt.Println("[ABORT] version not in manifest: ", ver)
		os.Exit(1)
	}
	dd := datadir()
	basedir := path.Join(dd, "versions")
	fpath := ver
	wg := sync.WaitGroup{}
	filehash := func(path string) string {
		fd, err := os.Open(path)
		if err != nil {
			fmt.Println("[ABORT] Can't open file to hash it: ", err)
			os.Exit(1)
		}
		defer fd.Close()
		hasher := md5.New()
		if _, err := io.Copy(hasher, fd); err != nil {
			fmt.Println("[ABORT] Can't hash file: ", err)
			os.Exit(1)
		}
		sum := hex.EncodeToString(hasher.Sum(nil))
		return sum
	}
	get := func(filepath, hash string) {
		fmt.Println("[GET]", filepath)
		localfile := path.Join(basedir, filepath)
		resp, err := http.Get(REPO + filepath)
		if err != nil {
			fmt.Printf("[ERR] Could not download file(%s): %s\n", filepath, err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		err = os.MkdirAll(path.Dir(localfile), 0777)
		if err != nil {
			fmt.Printf("[ERR] Could create local file directory(%s): %s\n", localfile, err)
			os.Exit(1)
		}
		ofile, err := os.Create(localfile)
		if err != nil {
			fmt.Printf("[ERR] Could create local file(%s): %s\n", localfile, err)
			os.Exit(1)
		}
		io.Copy(ofile, resp.Body)
		ofile.Close()
		if filehash(localfile) != hash {
			fmt.Printf("[ERR] Downloaded file hash mismatch: %s\n", localfile)
			os.Exit(1)
		}
		wg.Done()
	}
	copyfile := func(dst, src string) error {
		in, err := os.Open(src)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		cerr := out.Close()
		if err != nil {
			return err
		}
		return cerr
	}
	latest_local := m.FindLatestLocal()
	tryOldFile := func(filepath, basedir, expectedhash string) bool {
		//TODO check hash and copy old file
		if latest_local == nil {
			return false
		}
		noverpath := filepath[strings.Index(filepath, "/"):]
		opath := path.Join(latest_local.Path, noverpath)
		_, err := os.Stat(opath)
		if err == nil {
			if filehash(opath) == expectedhash {
				//fmt.Printf("[CACHED] %s\n", filepath)
				err = os.MkdirAll(path.Dir(path.Join(basedir, filepath)), 0777)
				if err != nil {
					fmt.Printf("[ERR] Could not create parent dirs: %s\n", err)
					os.Exit(1)
				}
				err := copyfile(path.Join(basedir, filepath), opath)
				if err != nil {
					fmt.Printf("[ERR] Could not reuse old file: %s\n", err)
					os.Exit(1)
				}
				return true
			}
		}
		return false
	}
	for dirname, files := range entry {
		dirpath := path.Join(fpath, dirname)
		for _, filepair := range files {
			toks := strings.Split(filepair, ",")
			file := toks[0]
			hash := toks[1]
			//Check if it exists
			filepath := path.Join(dirpath, file)
			localfile := path.Join(basedir, filepath)
			_, err := os.Stat(localfile)
			if os.IsNotExist(err) {
				//TODO possibly copy from older version
				if !tryOldFile(filepath, basedir, hash) {
					wg.Add(1)
					go get(filepath, hash)
				}
				continue
			}
			if err != nil {
				fmt.Println("[ABORT] file exists but: ", err)
				os.Exit(1)
			}
			//File exists, check hash
			sum := filehash(localfile)
			if sum != hash {
				fmt.Println("[WARN] hash mismatch on", localfile)
				wg.Add(1)
				go get(filepath, hash)
			}
		}
	}
	wg.Wait()
}
func (m *Manifest) DownloadLatest() {
	m.Download(m.Latest())
}
func (m *Manifest) FindLatestLocal() *App {
	local := m.FindLocal()
	var rv *App = nil
	mmaj := 0
	mmin := 0
	mbuild := 0
	for _, l := range local {
		if l.Major > mmaj {
			rv = l
			mmaj = l.Major
			mmin = l.Minor
			mbuild = l.Build
		} else if l.Major == mmaj {
			if l.Minor > mmin {
				rv = l
				mmaj = l.Major
				mmin = l.Minor
				mbuild = l.Build
			} else if l.Minor == mmin {
				if l.Build > mbuild {
					rv = l
					mmaj = l.Major
					mmin = l.Minor
					mbuild = l.Build
				}
			}
		}
	}
	return rv
}
func precheck() {
	dd := datadir()
	basedir := path.Join(dd, "versions")
	fmt.Println("making: ", basedir)
	os.MkdirAll(basedir, 0777)
}
func main() {
	precheck()
	mf := TryGetManifest()
	if mf != nil {
		mf.DownloadLatest()
	}
	app := mf.FindLatestLocal()
	if app != nil {
		fmt.Printf("Launching %d.%d.%d\n", app.Major, app.Minor, app.Build)
		app.Launch()
	} else {
		fmt.Println("Could not download latest version, and no local version found")
	}

}
