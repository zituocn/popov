package popov

import (
	"regexp"
	"strconv"
)

// DirNode directory node
type DirNode struct {
	Title string // title text
	Link  string // name link
	Depth int    // tag depth
}

var (
	// hTagRegexp  h1~h6 tag regexp string
	hTagRegexp = regexp.MustCompile(`<h(\d) (.*?)>(.*?)</h(\d)>`)

	// link name attribute regexp string
	linkAttribRegexp = regexp.MustCompile(`<a name="(.*?)"|<a name='(.*?)'`)
)

// NewDirNode  returns []*DirNode
func NewDirNode(content string) (data []*DirNode) {
	hTags := hTagRegexp.FindAllString(content, -1)
	data = make([]*DirNode, 0, len(hTags))
	for _, s := range hTags {
		if s == "" {
			continue
		}
		data = append(data, &DirNode{
			Title: StripTags(s),
			Link:  getLink(s),
			Depth: getTagNumber(s),
		})
	}
	return data
}

func getLink(s string) string {
	ss := linkAttribRegexp.FindStringSubmatch(s)
	if len(ss) == 3 { // if regexp find
		if ss[1] != "" { // match "
			return ss[1]
		}
		if ss[2] != "" { // match '
			return ss[2]
		}
	}
	return ""
}

func getTagNumber(s string) int {
	if s == "" {
		return 0
	}
	if len(s) > 3 {
		i, _ := strconv.Atoi(s[2:3])
		return i
	}
	return 0
}
