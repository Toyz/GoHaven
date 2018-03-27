package GoHaven

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/mewkiz/pkg/errutil"
)

func New() *WallHaven {
	return &WallHaven{}
}

// Search searches for wallpapers based on the given query and search options.
func (wh *WallHaven) Search(query string, options ...Option) (ids []ID, err error) {
	// Parse search options.
	values := make(url.Values)
	if len(query) != 0 {
		values.Add("q", query)
	}
	for _, option := range options {
		key := option.Key()
		val := option.Value()
		values.Add(key, val)
	}

	// Send search request.
	rawquery := values.Encode()
	rawurl := "http://alpha.wallhaven.cc/search?" + rawquery
	doc, err := goquery.NewDocument(rawurl)
	if err != nil {
		return nil, errutil.Err(err)
	}

	// Locate wallpaper IDs in response.
	//
	// Example response:
	//    <figure id="thumb-109603" class="thumb thumb-sfw thumb-general" data-wallpaper-id="109603" style="width:300px;height:200px" >
	f := func(i int, s *goquery.Selection) {
		rawid, ok := s.Attr("data-wallpaper-id")
		if !ok {
			return
		}
		id, err := strconv.Atoi(rawid)
		if err != nil {
			log.Print(errutil.Err(err))
			return
		}
		ids = append(ids, ID(id))
	}
	doc.Find("figure.thumb").Each(f)

	return ids, nil
}

// ID represents the wallpaper ID of a specific wallpaper on wallhaven.cc.
type ID int

func (id ID) Details() (details *ImageDetail, err error) {
	rawurl := fmt.Sprintf("https://alpha.wallhaven.cc/wallpaper/%d", id)
	doc, err := goquery.NewDocument(rawurl)
	if err != nil {
		return nil, errutil.Err(err)
	}
	tags := make([]string, 0)

	f := func(i int, s *goquery.Selection) {
		tags = append(tags, s.Text())
	}
	doc.Find("a.tagname").Each(f)

	url, _ := doc.Find("img#wallpaper").Attr("src")

	return &ImageDetail{
		Tags: tags,
		URL:  fmt.Sprintf("https:%s", url),
	}, nil
}

func (detail *ImageDetail) Download(dir string) (path string, err error) {
	download := func(url string) (path string, err error) {
		parts := strings.Split("/", url)
		path = filepath.Join(dir, parts[len(parts)-1])

		resp, err := http.Get(url)
		if err != nil {
			return "", errutil.Err(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", errutil.Newf("invalid status code; expected %d, got %d", http.StatusOK, resp.StatusCode)
		}
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", errutil.Err(err)
		}
		if err := ioutil.WriteFile(path, buf, 0644); err != nil {
			return "", errutil.Err(err)
		}
		return path, nil
	}

	path, _ = download(detail.URL)
	if err != nil {
		return "", errutil.Err(err)
	}
	return path, nil
}
