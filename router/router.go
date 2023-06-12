package router

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/russross/blackfriday/v2"
)

const dataDir = "data/"
const headerTmplPath = dataDir + ".config/templates/header.tmpl.html"
const indexTmplPath = dataDir + ".config/templates/index.tmpl.html"
const postsTmplPath = dataDir + ".config/templates/posts.tmpl.html"
const postTmplPath = dataDir + ".config/templates/post.tmpl.html"

type Config struct {
	Port   string
	Title  string
	Header string
}

var config Config

func BlogServer() *chi.Mux {
	configJson, _ := os.ReadFile(dataDir + ".config/config.json")
	json.Unmarshal(configJson, &config)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(dataDir+".config/static/"))))

	r.Route("/", func(r chi.Router) {
		r.Get("/", getIndex)
		r.Get("/posts", getPosts)
		r.Get("/{pageName}", getPage)
		r.Get("/publish", getPublish)
		r.Get("/posts/{postName}", getPost)
		r.Get("/posts/{postName}/*", servePostAssets)
	})

	return r
	// http.ListenAndServe("127.0.0.1:"+config.Port, r)
}

func getIndex(w http.ResponseWriter, r *http.Request) {
	content, err := os.ReadFile(dataDir + ".pages/index.md")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	htmlContent := blackfriday.Run(content)
	htmlContent = []byte(replaceImagePaths(string(htmlContent), "index"))

	posts := getPostsFromDataSource()
	if len(posts) > 5 {
		posts = posts[:5]
	}

	tmpl, err := template.ParseFiles(
		headerTmplPath,
		indexTmplPath,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Title        string
		Header       string
		MarkdownHTML template.HTML
		Posts        []Post
	}{
		Title:        config.Title,
		Header:       config.Header,
		MarkdownHTML: template.HTML(htmlContent),
		Posts:        posts,
	}

	err = tmpl.ExecuteTemplate(w, "header.tmpl.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "index.tmpl.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getPosts(w http.ResponseWriter, r *http.Request) {
	posts := getPostsFromDataSource()

	tmpl, err := template.ParseFiles(
		headerTmplPath,
		postsTmplPath,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Title  string
		Header string
		Posts  []Post
	}{
		Title:  config.Title + " - Posts",
		Header: config.Header,
		Posts:  posts,
	}

	err = tmpl.ExecuteTemplate(w, "header.tmpl.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "posts.tmpl.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getPage(w http.ResponseWriter, r *http.Request) {
	pageName := chi.URLParam(r, "pageName")
	content, err := os.ReadFile(dataDir + ".pages/" + pageName + ".md")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	htmlContent := blackfriday.Run(content)
	htmlContent = []byte(replaceImagePaths(string(htmlContent), ".pages"))

	data := struct {
		Title        string
		Header       string
		MarkdownHTML template.HTML
	}{
		Title:        config.Title + " - " + pageName,
		Header:       config.Header,
		MarkdownHTML: template.HTML(htmlContent),
	}

	tmpl, err := template.ParseFiles(
		headerTmplPath,
		postTmplPath,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "header.tmpl.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "post.tmpl.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getPost(w http.ResponseWriter, r *http.Request) {
	postName := chi.URLParam(r, "postName")

	content, err := os.ReadFile(dataDir + postName + "/README.md")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	htmlContent := blackfriday.Run(content)
	htmlContent = []byte(replaceImagePaths(string(htmlContent), postName))

	tmpl, err := template.ParseFiles(
		headerTmplPath,
		postTmplPath,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	posts := getPostsFromDataSource()

	// find the post in the list of posts
	var post Post
	for _, p := range posts {
		if p.Path == "posts/"+postName {
			post = p
			break
		}
	}

	data := struct {
		Title        string
		Header       string
		MarkdownHTML template.HTML
	}{
		Title:        config.Header + " - " + post.Title,
		Header:       config.Header,
		MarkdownHTML: template.HTML(htmlContent),
	}

	err = tmpl.ExecuteTemplate(w, "header.tmpl.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "post.tmpl.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getPublish(w http.ResponseWriter, r *http.Request) {
	err := UpdatePosts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		w.Write([]byte("Published"))
	}
}

func servePostAssets(w http.ResponseWriter, r *http.Request) {
	postName := chi.URLParam(r, "postName")
	http.StripPrefix("/posts/"+postName+"/", http.FileServer(http.Dir(dataDir+postName+"/"))).ServeHTTP(w, r)
}

func getPostsFromDataSource() []Post {
	// read the posts list from json file
	postList, err := os.ReadFile(dataDir + "postsList.json")
	if err != nil {
		return []Post{}
	}
	// convert the json to []Post
	var posts []Post
	err = json.Unmarshal(postList, &posts)
	if err != nil {
		return []Post{}
	}

	return posts
}

func replaceImagePaths(htmlContent string, dirPath string) string {
	// Find all <img> tags in the HTML content
	imgTagStart := "<img"
	// imgTagEnd := ">"
	imgTags := strings.Split(htmlContent, imgTagStart)

	// Iterate over the <img> tags and update the src attribute
	for i := 1; i < len(imgTags); i++ {
		imgTag := imgTagStart + imgTags[i]

		// Find the src attribute within the <img> tag
		srcStartIndex := strings.Index(imgTag, "src=\"")
		if srcStartIndex == -1 {
			continue
		}
		srcEndIndex := strings.Index(imgTag[srcStartIndex+5:], "\"")
		if srcEndIndex == -1 {
			continue
		}

		// Extract the image filename from the src attribute
		src := imgTag[srcStartIndex+5 : srcStartIndex+5+srcEndIndex]

		// If the src is a url, skip it
		if strings.HasPrefix(src, "http") {
			continue
		}

		// Build the correct relative path to the image
		relImagePath := filepath.Join(dirPath, src)

		// Update the src attribute with the correct URL
		updatedImgTag := strings.Replace(imgTag, src, relImagePath, 1)

		// Replace the original <img> tag with the updated one
		htmlContent = strings.Replace(htmlContent, imgTagStart+imgTags[i], updatedImgTag, 1)
	}

	return htmlContent
}
