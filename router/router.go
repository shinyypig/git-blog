package router

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/russross/blackfriday/v2"
	"github.com/sosedoff/gitkit"
)

const dataDir = "data/"
const repoDir = "git/"
const headerTmplPath = dataDir + ".config/templates/header.tmpl.html"
const indexTmplPath = dataDir + ".config/templates/index.tmpl.html"
const postsTmplPath = dataDir + ".config/templates/posts.tmpl.html"
const postTmplPath = dataDir + ".config/templates/post.tmpl.html"
const postListPath = dataDir + ".pages/postsList.json"

var config Config

type Config struct {
	AnaylzePostsOnStart bool
	BlogHeader          string
	BlogTitle           string
	PostDefaultState    string
	GitPassword         string
	GitUserName         string
	WebPort             string
	WebIP               string
}

type MyGitServer struct {
	originalServer   *gitkit.Server
	additionalHander func(w http.ResponseWriter, r *http.Request)
}

func RunBlogServer() {
	configJson, _ := os.ReadFile(dataDir + ".config/config.json")
	json.Unmarshal(configJson, &config)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	gitServer := createGitServer()
	r.Route("/", func(r chi.Router) {
		r.Get("/", getIndex)
		r.Get("/{pageName}", getPage)
		r.Get("/posts/{postName}", getPost)
		r.Get("/posts/{postName}/*", servePostAssets)
		r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(dataDir+".config/static/"))))
		// git sevice
		r.Handle("/{gitName}/info/*", gitServer)
		r.Handle("/{gitName}/git-receive-pack", gitServer)
		r.Handle("/{gitName}/git-upload-pack", gitServer)
	})

	log.Println("Starting server on " + config.WebIP + ":" + config.WebPort)
	err := http.ListenAndServe(config.WebIP+":"+config.WebPort, r)
	if err != nil {
		log.Fatalln(err)
	}
}

func createGitServer() *MyGitServer {
	hooks := &gitkit.HookScripts{
		PreReceive: `echo "Hello World!"`,
	}

	originalServer := gitkit.New(gitkit.Config{
		Dir:        "git/",
		AutoCreate: true,
		AutoHooks:  true,
		Auth:       true,
		Hooks:      hooks,
	})

	originalServer.AuthFunc = func(cred gitkit.Credential, req *gitkit.Request) (bool, error) {
		log.Println("user auth request for repo:", cred.Username, cred.Password, req.RepoName)
		return cred.Username == config.GitUserName && cred.Password == config.GitPassword, nil
	}

	if err := originalServer.Setup(); err != nil {
		log.Fatal(err)
	}

	return &MyGitServer{
		originalServer:   originalServer,
		additionalHander: gitUpdate,
	}
}

func (s *MyGitServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.originalServer.ServeHTTP(w, r)
	s.additionalHander(w, r)
}

func gitUpdate(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "git-receive-pack") {
		gitName := chi.URLParam(r, "gitName")
		log.Printf("git-receive-pack: %s", gitName)
		extractGitData(gitName)
		updatePost(gitName, posts)
	}
}

func getIndex(w http.ResponseWriter, r *http.Request) {
	content, err := os.ReadFile(dataDir + ".pages/index.md")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	htmlContent := blackfriday.Run(content)
	htmlContent = []byte(replaceImagePaths(string(htmlContent), "index"))

	recentPosts := publicPosts
	if len(recentPosts) > 5 {
		recentPosts = recentPosts[:5]
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
		Title:        config.BlogTitle,
		Header:       config.BlogHeader,
		MarkdownHTML: template.HTML(htmlContent),
		Posts:        recentPosts,
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

func getPage(w http.ResponseWriter, r *http.Request) {
	pageName := chi.URLParam(r, "pageName")
	if pageName == "posts" {
		getPosts(w, r)
		return
	}

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
		Title:        config.BlogTitle + " - " + pageName,
		Header:       config.BlogHeader,
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

func getPosts(w http.ResponseWriter, r *http.Request) {
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
		Title:  config.BlogTitle + " - Posts",
		Header: config.BlogHeader,
		Posts:  publicPosts,
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

func getPost(w http.ResponseWriter, r *http.Request) {
	postName := chi.URLParam(r, "postName")

	// check if the post is public
	postInfo, err := getPostInfo(postName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if postInfo.State != "public" {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

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

	// find the post in the list of posts
	var post Post
	for _, p := range posts {
		if p.Name == postName {
			post = p
			break
		}
	}

	data := struct {
		Title        string
		Header       string
		MarkdownHTML template.HTML
	}{
		Title:        config.BlogHeader + " - " + post.Title,
		Header:       config.BlogHeader,
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

func servePostAssets(w http.ResponseWriter, r *http.Request) {
	postName := chi.URLParam(r, "postName")
	http.StripPrefix("/posts/"+postName+"/", http.FileServer(http.Dir(dataDir+postName+"/"))).ServeHTTP(w, r)
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
