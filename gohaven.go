package GoHaven

import (
	"fmt"
	"io"
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

func (wh *WallHaven) processHTML(url string) (results *SearchInfo, err error) {
	searchInfo := &SearchInfo{
		Results: make([]SearchResult, 0),
	}

	doc, err := goquery.NewDocument(url)

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
		searchInfo.Results = append(searchInfo.Results, SearchResult{
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

	pageString := doc.Find("header.thumb-listing-page-header").Find("h2").Text()
	pageString = strings.Replace(pageString, "Page ", "", 1)
	pageValues := strings.Split(pageString, " / ")
	if len(pageValues) > 1 {
		searchInfo.CurrentPage, _ = strconv.Atoi(strings.TrimSpace(pageValues[0]))
		searchInfo.TotalPages, _ = strconv.Atoi(strings.TrimSpace(pageValues[1]))
	} else {
		searchInfo.CurrentPage = 1
		searchInfo.TotalPages = 1
	}

	searchInfo.End = searchInfo.CurrentPage == searchInfo.TotalPages
	return searchInfo, nil
}

// Search searches for wallpapers based on the given query and search options.
func (wh *WallHaven) Search(query string, options ...Option) (result *SearchInfo, err error) {
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

	return wh.processHTML(rawurl)
}

func (wh *WallHaven) UserUploads(user string, options ...Option) (result *SearchInfo, err error) {
	values := make(url.Values)

	for _, option := range options {
		if option.Key() == "purity" {
			key := option.Key()
			val := option.Value()
			values.Add(key, val)
		}

		if option.Key() == "page" {
			key := option.Key()
			val := option.Value()
			values.Add(key, val)
		}
	}

	return wh.processHTML(fmt.Sprintf("https://alpha.wallhaven.cc/user/%s/uploads?%s", user, values.Encode()))
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
		r, g, b := HexToRGB(style)

		colors = append(colors, Color{
			HEX:  style,
			RGB:  fmt.Sprintf("%s,%s,%s", strconv.Itoa(r), strconv.Itoa(g), strconv.Itoa(b)),
			Link: link,
		})
	})

	url, _ := doc.Find("img#wallpaper").Attr("src")
	uploadedOn, _ := doc.Find("[datetime]").Attr("datetime")

	var purtity string
	/*
		data := doc.Find("#wallpaper-purity-form input[checked]")
		data = data.Prev()

		n := data.Get(0) // Retrieves the internal *html.Node
		for _, a := range n.Attr {
			fmt.Printf("%s=%s\n", a.Key, a.Val)
		}

		log.Println(data.Parent().Attr("class"))
		log.Println(data.Parent().Attr("id"))
	*/
	doc.Find("#wallpaper-purity-form").Children().Each(func(i int, s *goquery.Selection) {
		if s.Is("label") {
			radio := s.Next()
			if radio.Is("[checked]") {
				log.Println(s.Text())
			}
		}
	})

	var gallery string
	var views int
	var favorites int
	doc.Find("div[data-storage-id='showcase-info'] > dl").Children().Each(func(i int, s *goquery.Selection) {
		if s.Is("dt") {
			title := s.Text()
			if title == "Category" {
				gallery = s.Next().Text()
				return
			}

			if title == "Views" {
				v := s.Next().Text()
				views, _ = strconv.Atoi(strings.Replace(v, ",", "", -1))
				return
			}

			if title == "Favorites" {
				v := s.Next().Text()
				favorites, _ = strconv.Atoi(strings.Replace(v, ",", "", -1))
				return
			}
		}
	})

	uploaderInfo := doc.Find("a.username")
	uploaderProfileImage := doc.Find("a.avatar > img")
	pImage, _ := uploaderProfileImage.Attr("src")
	uploader.Name = uploaderInfo.Text()
	uploader.Profile, _ = uploaderInfo.Attr("href")
	uploader.ProfilePicture = fmt.Sprintf("https:%s", pImage)

	return &ImageDetail{
		Tags:       tags,
		URL:        fmt.Sprintf("https:%s", url),
		Views:      views,
		Category:   gallery,
		Purity:     purtity,
		Favorites:  favorites,
		Uploader:   uploader,
		UploadedOn: uploadedOn,
		ImageID:    id,
		Colors:     colors,
		Link:       rawurl,
	}, nil
}

func (detail *ImageDetail) GetImageBuffer() (io.ReadCloser, error) {
	resp, err := http.Get(detail.URL)
	if err != nil {
		return nil, errutil.Err(err)
	}
	return resp.Body, nil
}

func (detail *ImageDetail) ReadAll() (buf []byte, err error) {
	resp, err := http.Get(detail.URL)
	if err != nil {
		return nil, errutil.Err(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errutil.Newf("invalid status code; expected %d, got %d", http.StatusOK, resp.StatusCode)
	}
	buf, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errutil.Err(err)
	}
	return
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
