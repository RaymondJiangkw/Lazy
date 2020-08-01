// Fetch fetch the content found at a URL with cache.

package extract

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

// Have homePath. Form: `homePath`/`cacheFolder`/`fileName`
var id2file map[string]string = make(map[string]string)

var homePath string

// Whether caching mechanism can work properly.
var cacheWorking bool = true

const (
	notifyWhenNotWorking = "Cache Optimization is abandoned. Switch to download-instantly mode."
	cacheFolder          = "html"
	cacheHtmlFormat      = "%d_%s"
	cacheMaximumSize     = 1024 * 1024 * 256 // 256 MiB
	fetchMaximumTry      = 5
	pauseSeconds         = 5
	maximumRoutines      = 5
	redirectScript       = "window.location="
)

// Init check the existence of caching folder, "html".
func init() {
	// Get `homePath`
	var ok error
	if homePath, ok = os.Getwd(); ok != nil {
		fmt.Fprintf(os.Stderr, "Encounter Error %v while getting homePath.\n%s\n", ok, notifyWhenNotWorking)
		cacheWorking = false
	} else {
		homePath = path.Join(homePath, ".cache")
		if _, ok = os.Stat(homePath); os.IsNotExist(ok) {
			os.Mkdir(homePath, 0777)
		}
		if _, ok = os.Stat(homePath); ok != nil {
			fmt.Fprintf(os.Stderr, "Encounter Error %v while getting cachePath.\n%s\n", ok, notifyWhenNotWorking)
			cacheWorking = false
		}
	}
	// Check the existence of `cacheFolder`
	if cacheWorking {
		if _, err := os.Stat(path.Join(homePath, cacheFolder)); err != nil {
			if os.IsNotExist(err) {
				if err := os.Mkdir(path.Join(homePath, cacheFolder), 0777); err != nil {
					fmt.Fprintf(os.Stderr, "Encounter Error %v while creating cacheFolder.\n%s\n", err, notifyWhenNotWorking)
					cacheWorking = false
				}
			} else {
				fmt.Fprintf(os.Stderr, "Encounter Error %v while checking stat of cacheFolder.\n%s\n", err, notifyWhenNotWorking)
				cacheWorking = false
			}
		}
	}
	// Read `cache`
	if cacheWorking {
		if cacheFolderHandle, ok := os.Open(path.Join(homePath, cacheFolder)); ok != nil {
			fmt.Fprintf(os.Stderr, "Encounter Error %v while opening cacheFolder.\n%s\n", ok, notifyWhenNotWorking)
			cacheWorking = false
		} else {
			files, ok := cacheFolderHandle.Readdir(-1)
			if ok != nil {
				fmt.Fprintf(os.Stderr, "Encounter Error %v while reading files inside cacheFolder.\n%s\n", ok, notifyWhenNotWorking)
				cacheWorking = false
			} else {
				var totalSize int64
				var id string
				var timeUnix int64
				for _, file := range files {
					fileFullPath := path.Join(homePath, cacheFolder, file.Name())
					if file.Mode().IsRegular() {
						fmt.Sscanf(file.Name(), cacheHtmlFormat, &timeUnix, &id)
						totalSize += file.Size()
						id2file[id] = fileFullPath
					} else {
						os.Remove(fileFullPath)
					}
				}
				if totalSize > cacheMaximumSize {
					fmt.Printf("Size of Cache Files exceeds %d Bytes. Cleaning...", cacheMaximumSize)
					if ok = os.RemoveAll(path.Join(homePath, cacheFolder)); ok != nil {
						fmt.Fprintf(os.Stderr, "Encounter Error %v while cleaning cache files.\n%s\n", ok, notifyWhenNotWorking)
						cacheWorking = false
					} else {
						if ok = os.Mkdir(path.Join(homePath, cacheFolder), 0777); ok != nil {
							fmt.Fprintf(os.Stderr, "Encounter Error %v while creating cacheFolder.\n%s\n", ok, notifyWhenNotWorking)
							cacheWorking = false
						} else {
							// Refresh `id2file`
							id2file = make(map[string]string)
						}
					}
				}
			}
		}
	}
}

// encoding determine for html page , eg: gbk gb2312 GB18030
func determineEncoding(r io.Reader) (encoding.Encoding, error) {
	bytes, err := bufio.NewReader(r).Peek(1024)
	if err != nil {
		return nil, err
	}
	e, _, _ := charset.DetermineEncoding(bytes, "")
	return e, nil
}

func fetch(url string, errWriter io.Writer) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(errWriter, "Download page: %s error with status code %d.\n", url, resp.StatusCode)
		err = fmt.Errorf("Status Code Error %d", resp.StatusCode)
		return "", err
	}
	// Solve the Encoding Issue
	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	e, err := determineEncoding(bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	reader := transform.NewReader(bytes.NewReader(raw), e.NewDecoder())
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	content := string(b)
	// Solve window.location
	if redirLoc := strings.Index(content, redirectScript); redirLoc != -1 {
		newURL := content[redirLoc+len(redirectScript):]
		sep := newURL[:1]
		newURL = newURL[1:][:strings.Index(newURL[1:], sep)]
		newURL, err = completeURL(url, newURL)
		if err != nil {
			return content, nil
		}
		return fetch(newURL, errWriter)
	}
	return content, nil
}

// id give nearly unique id for every url
// id does not cache the result, since `url` is usually very short.
func id(url string) (id string) {
	h := sha1.New()
	h.Write([]byte(url))
	return hex.EncodeToString(h.Sum(nil))
}

func fileName(url string) (name string, err error) {
	var buf bytes.Buffer
	_, err = fmt.Fprintf(&buf, cacheHtmlFormat, time.Now().Unix(), id(url))
	if err != nil {
		return "", err
	}
	return buf.String(), err
}

func fetchWithTry(url string, errWriter io.Writer) (data string, err error) {
	for i := 0; i < fetchMaximumTry; i++ {
		if i != 0 {
			fmt.Fprintf(errWriter, "Encounter Error %v while fetching url %s. Retry the %d th time. Pause %d secs.\n", err, url, i, pauseSeconds)
			time.Sleep(time.Second * pauseSeconds)
		}
		data, err = fetch(url, errWriter)
		if err == nil {
			break
		}
	}
	if err != nil {
		fmt.Fprintf(errWriter, "Retry Times Out.\n")
	}
	return data, err
}

func normalizeURL(url string) string {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	} else {
		return "http://" + url
	}
}

func fetchOneURL(url string, refresh bool, errWriter io.Writer) (string, error) {
	url = normalizeURL(url)
	if !cacheWorking {
		return fetchWithTry(url, errWriter)
	}
	// Check Cache
	if filePath, ok := id2file[id(url)]; ok && !refresh {
		// Have Available Cache
		if file, err := os.Open(filePath); err == nil {
			if body, err := ioutil.ReadAll(file); err == nil {
				return string(body), nil
			} else {
				fmt.Fprintf(errWriter, "Encounter error %v while reading cache file, deleting cache file...\n", err)
				os.Remove(filePath)
			}
		} else {
			fmt.Fprintf(errWriter, "Encounter error %v while opening cache file, deleting cache file...\n", err)
			os.Remove(filePath)
		}
	}
	body, err := fetchWithTry(url, errWriter)
	if err != nil {
		return body, err
	} else {
		// Write Cache File
		cacheFileName, err := fileName(url)
		if err == nil {
			var writeSuccess bool = false
			file, err := os.Create(path.Join(homePath, cacheFolder, cacheFileName))
			if err == nil {
				_, err := file.WriteString(body)
				if err == nil {
					id2file[id(url)] = path.Join(homePath, cacheFolder, cacheFileName)
					writeSuccess = true
				}
			}
			file.Close()
			if !writeSuccess {
				os.Remove(path.Join(homePath, cacheFolder, cacheFileName))
			}
		}
		return body, nil
	}
}

func Fetch(refresh bool, errWriter io.Writer, info chan<- int, urls ...string) (data []string, errs []error) {
	data = make([]string, len(urls), len(urls))
	errs = make([]error, len(urls), len(urls))
	var tokens = make(chan struct{}, maximumRoutines)
	var wg sync.WaitGroup
	var finish int
	var lock sync.Mutex
	for i, url := range urls {
		wg.Add(1)
		go func(i int, url string) {
			defer func() {
				lock.Lock()
				finish++
				if info != nil {
					info <- finish
				}
				lock.Unlock()
				wg.Done()
			}()
			tokens <- struct{}{} // Acquire a token
			_data, _err := fetchOneURL(url, refresh, errWriter)
			data[i], errs[i] = _data, _err
			<-tokens // Release the token
		}(i, url)
	}
	wg.Wait()
	return
}
