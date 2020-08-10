// catalogue extract the catalogue of novel from its HTML file.
package extract

import (
	"bufio"
	"path"
	"strings"

	"github.com/RaymondJiangkw/Lazy/utils"
	"golang.org/x/net/html"
)

type Chapter struct {
	Name    string
	Url     string
	Content string
	Fetch   bool
}

type Chapters []*Chapter

func (c *Chapter) String() string {
	return "Name: " + c.Name + " Url: " + c.Url
}

func (c *Chapter) Equal(c_a *Chapter) bool {
	return c.Name == c_a.Name
}

// Catalogue give the catalogue in the url.
// It takes the method of getting <a> Tags under <dl>.
// However, not all websites use this mechanism. So, there is another
// method of getting the most <a> Tags under a <div> Tag.
func Catalogue(url string) (c Chapters, e error) {
	// NOTICE: This is a brute action to speed up.
	// We only accept folder or `index` here.
	if strings.ToLower(utils.PageNameURL(url)) != "index" && strings.Index(path.Base(url), ".") != -1 {
		return nil, utils.Invalid
	}

	bodies, errs, ioCompletes := utils.Fetch([]string{url}, &utils.FetchOption{Redirect: true, Refresh: true})
	defer utils.WaitSync(ioCompletes) // Sync I/O here, since `catalogue` only has one page.
	if errs[0] != nil {
		e = errs[0]
		return
	}

	doc, e := html.Parse(strings.NewReader(*bodies[0]))
	if e != nil {
		return
	}

	var aTags []utils.TagA
	if aTags = utils.ParseATags(extractAUnderDL(doc)); len(aTags) > 0 {
		// Method 1
		// Find <a> under <dl>
	} else if aTags = utils.ParseATags(extractAUnderUL(doc)); len(aTags) > 0 {
		// Method 2
		// Find <a> under <ul>
	} else if aTags = utils.ParseATags(mostAUnderDiv(doc)); len(aTags) > 0 {
		// Method 3
		// Find the most <a> under <div>
	} else {
		return nil, utils.Invalid
	}

	var tmp Chapters // Remove possible head duplications
	var exists map[string]int = make(map[string]int)
	for _, a := range aTags {
		if a.Href == "" || a.Text == "" {
			continue
		}
		url, err := utils.CompleteURL(url, a.Href)
		if err != nil {
			continue
		}
		tmp = append(tmp, &Chapter{Name: strings.TrimSpace(a.Text), Url: url})
		exists[strings.TrimSpace(a.Text)]++
	}
	// NOTICE: Remove Head Duplication(up-to-date chapters) and Tail Duplication(duplication).
	// Switch to only record the last time it appears. =_= Some websites have really bad format.
	// HeadDuplicate := true
	// appears := make(map[string]bool)
	for i := 0; i < len(tmp); i++ {
		/*
			if HeadDuplicate {
				if exists[tmp[i].Name] > 1 {
					exists[tmp[i].Name]--
					continue
				} else {
					HeadDuplicate = false
				}
			}
			if !appears[tmp[i].Name] {
				c = append(c, tmp[i])
				appears[tmp[i].Name] = true
			}
		*/
		if exists[tmp[i].Name] == 1 {
			c = append(c, tmp[i])
		}
		exists[tmp[i].Name]--
	}
	return
}

func extractAUnderDL(root *html.Node) []*html.Node {
	return utils.Select(root, "dl a")
}

func extractAUnderUL(root *html.Node) []*html.Node {
	return utils.Select(root, "ul a")
}

func mostAUnderDiv(root *html.Node) (ret []*html.Node) {
	tags := utils.Select(root, "div")
	avoidFunc := utils.SelectTagNames([]string{"div"})
	selectFunc := utils.SelectTagNames([]string{"a"})
	for _, tag := range tags {
		c := utils.Extract(tag, selectFunc, &utils.ExtractOption{IsAvoid: avoidFunc})
		if len(c) > len(ret) {
			ret = c
		}
	}
	return
}

func ValidCatalog(c_s []Chapters) []Chapters {
	if len(c_s) <= 1 {
		return c_s
	}
	// Prepare Partition
	nameGroups := make([][]string, len(c_s), len(c_s))
	data := make([]*utils.StringSlices, len(c_s), len(c_s))
	Names2Chapter := make(map[*[]string]Chapters)
	for i, c := range c_s {
		nameGroups[i] = make([]string, len(c), len(c))
		for j, d := range c {
			nameGroups[i][j] = d.Name
		}
		data[i] = (*utils.StringSlices)(&nameGroups[i])
		Names2Chapter[&nameGroups[i]] = c
	}
	groups, _ := utils.PartitionStringSlices(data, 0.5)
	// Find Appropriate Group
	maximumGroupIndex := 0
	var cntLength = func(d []*utils.StringSlices) int {
		ret := 0
		for _, _d := range d {
			ret += len(*_d)
		}
		return ret
	}
	for i := 1; i < len(groups); i++ {
		if len(groups[i]) > len(groups[maximumGroupIndex]) {
			maximumGroupIndex = i
		} else if len(groups[i]) == len(groups[maximumGroupIndex]) { // NOTICE: prefer longer urls.
			if cntLength(groups[i]) > cntLength(groups[maximumGroupIndex]) {
				maximumGroupIndex = i
			}
		}
	}
	// Reconstruct Output
	rets := make([]Chapters, len(groups[maximumGroupIndex]), len(groups[maximumGroupIndex]))
	for i := 0; i < len(groups[maximumGroupIndex]); i++ {
		rets[i] = Names2Chapter[(*[]string)(groups[maximumGroupIndex][i])]
	}
	return rets
}

func contentQualityOver(u, v *Chapter) *Chapter {
	const ratio = 0.8
	const minimumTextLength = 10
	// Test 1: Length. We prefer longer length.
	if float64(len(u.Content))/float64(len(v.Content)) < ratio {
		return v
	} else if float64(len(v.Content))/float64(len(u.Content)) < ratio {
		return u
	}
	// Test 2: Short Lines. We prefer less short lines.
	shortLineNumbers := func(s *string) int {
		ret := 0
		r := bufio.NewScanner(strings.NewReader(*s))
		for r.Scan() {
			if len(r.Text()) < minimumTextLength {
				ret++
			}
		}
		return ret
	}
	uShortLineNumbers := shortLineNumbers(&u.Content)
	vShortLineNumbers := shortLineNumbers(&v.Content)
	if uShortLineNumbers > vShortLineNumbers {
		return v
	} else if vShortLineNumbers > uShortLineNumbers {
		return u
	}
	// Test 3: Weird Symbols. We prefer less weird symbols.
	symbols := []string{"&", ";", "(", ")", "~", "@", "#", "%", "^", "*", "-", "+", "http", ":", "/", "<", ">"}
	symbolsCount := func(s *string) int {
		ret := 0
		for _, symbol := range symbols {
			ret += strings.Count(*s, symbol)
		}
		return ret
	}
	uSymbolsNum := symbolsCount(&u.Content)
	vSymbolsNum := symbolsCount(&v.Content)
	if uSymbolsNum > vSymbolsNum {
		return v
	} else if vSymbolsNum > uSymbolsNum {
		return u
	}
	// Otherwise, return the first.
	return u
}

func MergeCatalog(u, v Chapters) (rets Chapters) {
	// Special Cases
	if u == nil {
		return v
	}
	if v == nil {
		return u
	}
	uNameMap := make(map[string]*Chapter)
	vNameMap := make(map[string]*Chapter)
	uNames := make([]string, len(u), len(u))
	vNames := make([]string, len(v), len(v))
	for _, c := range u {
		uNameMap[c.Name] = c
		uNames = append(uNames, c.Name)
	}
	for _, c := range v {
		vNameMap[c.Name] = c
		vNames = append(vNames, c.Name)
	}
	mergedNames := utils.IntegrateStringSlices(uNames, vNames)
	for _, n := range mergedNames {
		c_1, ok_1 := uNameMap[n]
		c_2, ok_2 := vNameMap[n]
		// Case 1 only One Has
		if ok_1 == true && ok_2 == false {
			rets = append(rets, c_1)
		} else if ok_1 == false && ok_2 == true {
			rets = append(rets, c_2)
		}
		// Case 2 both Have
		if ok_1 && ok_2 {
			rets = append(rets, contentQualityOver(c_1, c_2))
		}
	}
	return
}
