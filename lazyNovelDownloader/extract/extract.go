// extract extract fiction from website in the format of []Chapter
package extract

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"time"
)

const (
	outputPrefixEachTurn = "	"
	extractCacheFolder   = ".novel"
	ProgressBarWidth     = 20
)

var extractCacheHome string

func generateString(length int, s string) string {
	var ret string
	for i := 0; i < length; i++ {
		ret += s
	}
	return ret
}

func rString(length int) (ret string) {
	ret = generateString(length, "\r")
	return
}

func spinner(prefix string, delay time.Duration, signal <-chan struct{}) {
	for {
		for _, r := range `-\|/` {
			select {
			case <-signal:
				return
			default:
				fmt.Printf("%s%s%c", rString(len(prefix)+1), prefix, r)
				time.Sleep(delay)
			}
		}
	}
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func ProgressBar(prefix string, defaultText string, maximum int, width int, delay time.Duration, signal <-chan struct{}, info <-chan int) {
	var lastText, currentText string
	var progress int
	var expectedRemainingTime time.Duration
	beginTime := time.Now()
	for {
		fmt.Printf("%s", rString(len(lastText)))
		select {
		case <-signal:
			fmt.Printf("%s", generateString(len(lastText)*2, " "))
			fmt.Printf("%s", rString(len(lastText)*2))
			return
		case i := <-info:
			progress = int((float64(i) / float64(maximum)) * float64(width))
			expectedRemainingTime = time.Duration(int64((float64(time.Since(beginTime)) / float64(i)) * float64(maximum-i)))
			currentText = strconv.Itoa(i) + "/" + strconv.Itoa(maximum) + " " + "[" + generateString(progress, ">") + generateString(width-progress, "-") + "]" + " Remaining: " + fmtDuration(expectedRemainingTime)
			defaultText = currentText
		default:
			currentText = defaultText
		}
		fmt.Printf("%s", prefix+currentText)
		lastText = prefix + currentText
	}
}

func Extract(url string, novelName string) (c_s Chapters, err error) {
	fmt.Printf("Fetching catalogue...\n")
	var signal chan struct{} = make(chan struct{}, 0)
	var progress chan int = make(chan int)
	go spinner("", 100*time.Millisecond, signal)
	c_s, err = Catalogue(url)
	signal <- struct{}{}
	if err != nil {
		return
	}
	fmt.Printf("\rDetected %d chapters from %s.\n", len(c_s), url)
	beginTime := time.Now()
	var urls []string
	var index, sum, times int
	for {
		times++
		for _, c := range c_s {
			if !c.Fetch {
				readFromCache := getFromCache(novelName, c.Name)
				if readFromCache != "" {
					c.Content = readFromCache
					c.Fetch = true
				} else {
					urls = append(urls, c.Url)
				}
			}
		}
		if len(urls) == 0 {
			break
		}
		fmt.Printf("%dth Turn:\n", times)
		contents, errors := Content(progress, func() {
			go ProgressBar(outputPrefixEachTurn+"Extract: ", "Extracting...", len(urls), ProgressBarWidth, time.Second, signal, progress)
		}, urls...)
		signal <- struct{}{}
		index, sum = 0, 0
		for _, c := range c_s {
			if !c.Fetch {
				if errors[index] == nil {
					c.Content = contents[index]
					c.Fetch = true
					writeToCache(novelName, c.Name, &c.Content)
				}
				index++
			}
			if c.Fetch {
				sum++
			}
		}
		urls = nil
		fmt.Printf(outputPrefixEachTurn+"Having complete %d pages.\n", sum)
		if sum != len(c_s) {
			fmt.Printf(outputPrefixEachTurn+"Pause %d secs for next turn.\n", pauseSeconds)
			time.Sleep(time.Second * pauseSeconds)
		}
	}
	fmt.Printf("After %dth Turn, finish all %d pages.\n", times-1, len(c_s))
	fmt.Printf("Total time: %.0f secs.\n", time.Since(beginTime).Seconds())
	return
}

func init() {
	home, err := os.Getwd()
	if err != nil {
		return
	}
	if _, err = os.Stat(path.Join(home, extractCacheFolder)); os.IsNotExist(err) {
		err = os.Mkdir(path.Join(home, extractCacheFolder), 0777)
		if err != nil {
			return
		}
	} else if err != nil {
		return
	}
	extractCacheHome = path.Join(home, extractCacheFolder)
}

func getFromCache(name string, chapter string) string {
	if extractCacheHome == "" {
		return ""
	}
	if _, err := os.Stat(path.Join(extractCacheHome, name)); err != nil {
		return ""
	}
	if _, err := os.Stat(path.Join(extractCacheHome, name, id(chapter))); err != nil {
		return ""
	}
	file, err := os.Open(path.Join(extractCacheHome, name, id(chapter)))
	if err != nil {
		return ""
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func writeToCache(name string, chapter string, content *string) {
	if extractCacheHome == "" {
		return
	}
	if _, err := os.Stat(path.Join(extractCacheHome, name)); os.IsNotExist(err) {
		err = os.Mkdir(path.Join(extractCacheHome, name), 0777)
		if err != nil {
			return
		}
	} else if err != nil {
		return
	}
	if _, err := os.Stat(path.Join(extractCacheHome, name, id(chapter))); err == nil {
		return
	} else if !os.IsNotExist(err) {
		os.Remove(path.Join(extractCacheHome, name, id(chapter)))
	}
	file, err := os.Create(path.Join(extractCacheHome, name, id(chapter)))
	defer file.Close()
	if err != nil {
		return
	}
	_, err = file.WriteString(*content)
	if err != nil {
		os.Remove(path.Join(extractCacheHome, name, id(chapter)))
	}
}
