package GoHaven

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
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
func (wh *WallHaven) Search(query string, options ...Option) (ids []SearchResult, err error) {
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
	rawurl := "https://alpha.wallhaven.cc/search?" + rawquery
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

		Purity := ""
		if s.HasClass("thumb-sfw") {
			Purity = "sfw"
		}

		if s.HasClass("thumb-sketchy") {
			Purity = "sketchy"
		}

		if s.HasClass("thumb-nsfw") {
			Purity = "nsfw"
		}

		category := ""
		if s.HasClass("thumb-anime") {
			category = "anime"
		}

		if s.HasClass("thumb-people") {
			category = "people"
		}

		if s.HasClass("thumb-general") {
			category = "general"
		}

		imageURL, _ := s.Find("img").Attr("data-src")
		thumbInfo := s.Find("div.thumb-info")

		widthXheight := strings.Split(thumbInfo.Find("span.wall-res").Text(), "x")
		width, _ := strconv.Atoi(strings.TrimSpace(widthXheight[0]))
		height, _ := strconv.Atoi(strings.TrimSpace(widthXheight[1]))

		favStringData := thumbInfo.Find("a.wall-favs").Text()
		favorites, _ := strconv.Atoi(favStringData)

		link, _ := s.Find("a.preview").Attr("href")
		ids = append(ids, SearchResult{
			ImageID:   ID(id),
			Thumbnail: imageURL,
			Purity:    Purity,
			Category:  category,
			Width:     width,
			Height:    height,
			Favorites: favorites,
			Link:      link,
		})
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
	tags := make([]Tag, 0)
	colors := make([]Color, 0)
	uploader := Uploader{}

	doc.Find("a.tagname").Each(func(i int, s *goquery.Selection) {
		Purity := ""
		parent := s.Closest("li.tag")

		if parent.HasClass("tag-sfw") {
			Purity = "sfw"
		}

		if parent.HasClass("tag-sketchy") {
			Purity = "sketchy"
		}

		if parent.HasClass("tag-nsfw") {
			Purity = "nsfw"
		}

		tidS, _ := parent.Attr("data-tag-id")
		tid, _ := strconv.Atoi(tidS)
		link, _ := s.Attr("href")

		tags = append(tags, Tag{
			Name:   s.Text(),
			Purity: Purity,
			TagID:  tid,
			Link:   link,
		})
	})

	doc.Find("li.color").Each(func(i int, s *goquery.Selection) {
		style, _ := s.Attr("style")

		style = strings.Replace(style, "background-color:", "", 1)
		link, _ := s.Find("a").Attr("href")

		colors = append(colors, Color{
			HEX:  style,
			Link: link,
		})
	})

	url, _ := doc.Find("img#wallpaper").Attr("src")
	uploadedOn, _ := doc.Find("[datetime]").Attr("datetime")

	uploaderInfo := doc.Find("a.username")
	uploaderProfileImage := doc.Find("a.avatar > img")
	pImage, _ := uploaderProfileImage.Attr("src")
	uploader.Name = uploaderInfo.Text()
	uploader.Profile, _ = uploaderInfo.Attr("href")
	uploader.ProfilePicture = fmt.Sprintf("https:%s", pImage)

	return &ImageDetail{
		Tags:       tags,
		URL:        fmt.Sprintf("https:%s", url),
		Uploader:   uploader,
		UploadedOn: uploadedOn,
		ImageID:    id,
		Colors:     colors,
		Link:       rawurl,
	}, nil
}

func (detail *ImageDetail) Download(dir string) (p string, err error) {
	download := func(url string) (p string, err error) {
		p = filepath.Join(dir, path.Base(url))

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
		if err := ioutil.WriteFile(p, buf, 0644); err != nil {
			return "", errutil.Err(err)
		}
		return p, nil
	}

	p, _ = download(detail.URL)
	if err != nil {
		return "", errutil.Err(err)
	}
	return p, nil
}
