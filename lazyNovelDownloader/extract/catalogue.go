// catalogue extract the catalogue of novel from its HTML file.
package extract

import (
	"io/ioutil"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type Chapter struct {
	Name    string
	Url     string
	Content string
	Fetch   bool
}

type aTag struct {
	href string
	text string
}

type Chapters []*Chapter
type Candidates []string

var invalidTextTags Candidates

func init() {
	invalidTextTags = Candidates([]string{"script", "style"})
}

// Catalogue give the catalogue in the url.
// It takes the method of getting <a> Tags under <dl>.
// However, not all websites use this mechanism. So, there is another
// method of getting the most <a> Tags under a <div> Tag.
func Catalogue(url string) (c Chapters, e error) {
	bodies, errs := Fetch(true, ioutil.Discard, nil, url)
	if errs[0] != nil {
		e = errs[0]
		return
	}
	doc, e := html.Parse(strings.NewReader(bodies[0]))
	if e != nil {
		return
	}
	var aTags []aTag
	var exists map[string]int = make(map[string]int)

	if aTags = parseATags(visitUnderNodeDFS(Candidates([]string{"a"}), nil, doc, Candidates([]string{"dl"}))); len(aTags) > 0 {
		// Method 1
		// Find <a> under <dl>
	} else if aTags = parseATags(mostUnderNode(Candidates([]string{"a"}), nil, doc, Candidates([]string{"div"}))); len(aTags) > 0 {
		// Method 2
		// Find the most <a> under <div>
	}
	var _c Chapters // Remove possible head duplications
	for _, a := range aTags {
		url, err := completeURL(url, a.href)
		if err != nil {
			continue
		}
		_c = append(_c, &Chapter{Name: strings.TrimSpace(a.text), Url: url})
		exists[strings.TrimSpace(a.text)]++
	}
	for _, ch := range _c {
		// Single
		if exists[ch.Name] == 1 {
			c = append(c, ch)
		}
		exists[ch.Name]--
	}
	return
}

func extractText(text string, n *html.Node, avoidTags Candidates, sep string) string {
	if n.Type == html.TextNode {
		text = text + sep + n.Data
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (avoidTags.indexOf(c.Data) != -1 || invalidTextTags.indexOf(c.Data) != -1) {
			continue
		}
		text = extractText(text, c, avoidTags, sep)
	}
	return text
}

func (c Candidates) indexOf(str string) int {
	for i, s := range c {
		if s == str {
			return i
		}
	}
	return -1
}

func extractTags(data Candidates, a []*html.Node, n *html.Node, avoidTags Candidates) []*html.Node {
	if n.Type == html.ElementNode && data.indexOf(n.Data) != -1 {
		a = append(a, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && avoidTags.indexOf(c.Data) != -1 {
			continue
		}
		a = extractTags(data, a, c, avoidTags)
	}
	return a
}

func completeURL(base string, appendix string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(appendix)
	if err != nil {
		return "", err
	}
	return b.ResolveReference(u).String(), nil
}

func visitUnderNodeDFS(targets Candidates, rets []*html.Node, n *html.Node, nodes Candidates) []*html.Node {
	if n.Type == html.ElementNode && nodes.indexOf(n.Data) != -1 {
		tag_s := extractTags(targets, nil, n, nil)
		rets = append(rets, tag_s...)
		return rets
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		rets = visitUnderNodeDFS(targets, rets, c, nodes)
	}
	return rets
}

// mostUnderNode only extract target tags, which are directly between nodes that satisfy the requirement.
func mostUnderNode(targets Candidates, rets []*html.Node, n *html.Node, nodes Candidates) []*html.Node {
	if n.Type == html.ElementNode && nodes.indexOf(n.Data) != -1 {
		tag_s := extractTags(targets, nil, n, nodes)
		if len(tag_s) > len(rets) {
			rets = tag_s
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		rets = mostUnderNode(targets, rets, c, nodes)
	}
	return rets
}

func parseATags(tags []*html.Node) (a_s []aTag) {
	for _, tag := range tags {
		for _, attr := range tag.Attr {
			if attr.Key == "href" {
				a_s = append(a_s, aTag{href: attr.Val, text: extractText("", tag, nil, "")})
				break
			}
		}
	}
	return
}
