// © 2017 the pinboard.adoc Authors under the WTFPL license. See AUTHORS for the list of authors.

// A Pinboard.in client that creates a single adoc index of your bookmarks.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	slugify "github.com/metal3d/go-slugify"
)

const (
	getPosts    = "https://api.pinboard.in/v1/posts/get"
	getAllPosts = "https://api.pinboard.in/v1/posts/all"
	getAllTags  = "https://api.pinboard.in/v1/tags/get"
)

var (
	authString = flag.String("auth", "", "Auth token for pinboard.in (required)")
	outFmt     = flag.String("fmt", "adoc", "Format for output. One of: [adoc,custom]")
	tmpl       = flag.String("tmpl", "", "Path to template file if fmt is custom")
	example    = flag.Bool("example", false, "Print out an example template and quit")
)

type postResp struct {
	Href        string    `json:"href"`
	Meta        string    `json:"meta"`
	Hash        string    `json:"hash"`
	Shared      string    `json:"shared"`
	Description string    `json:"description"`
	Extended    string    `json:"extended"`
	Time        time.Time `json:"time"`
	Tags        string    `json:"tags"`
	Toread      string    `json:"toread"`
	Slug        string
}

type auth string

func (a auth) buildUrl(inUrl string) *url.URL {
	newUrl, err := url.Parse(inUrl)
	if err != nil {
		log.Fatal(err)
	}
	q := newUrl.Query()
	q.Set("auth_token", string(a))
	q.Set("format", "json")
	newUrl.RawQuery = q.Encode()
	return newUrl
}

func main() {
	flag.Parse()

	if *example {
		fmt.Println(adocTemplate)
		return
	}

	if *authString == "" {
		fmt.Fprintln(os.Stderr, "You must provide an auth string.\n")
		flag.PrintDefaults()
		return
	}

	var t *template.Template
	if *outFmt == "adoc" {
		t = template.Must(template.New("adoc").Parse(adocTemplate))
	} else if *tmpl != "" {
		t = template.Must(template.New(*tmpl).ParseFiles(*tmpl))
	} else {
		fmt.Fprintln(os.Stderr, "You must provide a template path.\n")
		flag.PrintDefaults()
		return
	}

	a := auth(*authString)
	resp, err := http.Get(a.buildUrl(getAllPosts).String())
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != 200 {
		log.Fatalf("Response was: %s", resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var jsonResp []postResp
	err = json.Unmarshal(body, &jsonResp)
	if err != nil {
		log.Fatal(err)
	}

	byTag := map[string]postResp{}
	reading := []postResp{}
	for _, post := range jsonResp {
		post.Slug = slugify.Marshal(post.Description)
		tags := strings.Split(post.Tags, " ")
		for _, tag := range tags {
			byTag[tag] = post
		}

		if post.Toread == "yes" {
			reading = append(reading, post)
		}
	}

	err = t.Execute(os.Stdout, struct {
		Tags        map[string]postResp
		ReadingList []postResp
	}{
		Tags:        byTag,
		ReadingList: reading,
	})
	if err != nil {
		log.Fatal(err)
	}
}

var adocTemplate = `= Pinboard Links
:toc:
:toclevels: 1

{{if .ReadingList}}== Reading List{{end}}

{{range $item := .ReadingList}}
* <<{{$item.Slug}},{{$item.Description}}>>
{{end}}

{{range $key, $value := .Tags}}
{{if $key}}== {{ $key }}{{else}}== No Tags{{end}}

[#{{$value.Slug}}]
=== {{$value.Description}}

{{if $value.Extended}}{{$value.Extended}}{{end}}
Notes:: <<{{$value.Slug}}.adoc#>>
Date Collected:: {{$value.Time}}
Tags:: {{$value.Tags}}
To Read:: {{$value.Toread}}
{{end}}
`
