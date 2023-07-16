package router

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/sosedoff/gitkit"
	"github.com/sourcegraph/syntaxhighlight"
)

const dataDir = "data/"
const repoDir = "git/"
const tmplPath = dataDir + "_config/templates/*.tmpl.html"

var faviconFiles = map[string]string{
	"/favicon.ico":                "_config/static/favicon/favicon.ico",
	"/favicon-16x16.png":          "_config/static/favicon/favicon-16x16.png",
	"/favicon-32x32.png":          "_config/static/favicon/favicon-32x32.png",
	"/apple-touch-icon.png":       "_config/static/favicon/apple-touch-icon.png",
	"/android-chrome-192x192.png": "_config/static/favicon/android-chrome-192x192.png",
	"/android-chrome-512x512.png": "_config/static/favicon/android-chrome-512x512.png",
}

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
	extractGitData("_pages")
	extractGitData("_config")
	configJson, _ := os.ReadFile(dataDir + "_config/config.json")
	json.Unmarshal(configJson, &config)
	AnaylzePosts()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// include html templates folder

	gitServer := createGitServer()
	r.Route("/", func(r chi.Router) {
		r.Get("/", getIndex)
		r.Get("/{pageName}", getPage)
		r.Get("/posts/{postName}", getPost)
		r.Get("/posts/{postName}/*", servePostAssets)
		r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(dataDir+"_config/static/"))))
		r.Handle("/posts/_pages/*", http.StripPrefix("/posts/_pages/", http.FileServer(http.Dir(dataDir+"_pages/"))))
		// git sevice
		r.Handle("/{gitName}/info/*", gitServer)
		r.Handle("/{gitName}/git-receive-pack", gitServer)
		r.Handle("/{gitName}/git-upload-pack", gitServer)
		// favicon
		for route, path := range faviconFiles {
			r.Get(route, func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, dataDir+path)
			})
		}
	})

	log.Println("Starting server on " + config.WebIP + ":" + config.WebPort)
	err := http.ListenAndServe(config.WebIP+":"+config.WebPort, r)
	if err != nil {
		log.Fatalln(err)
	}
}

func createGitServer() *MyGitServer {
	hooks := &gitkit.HookScripts{
		PreReceive: `echo "Git Blog"`,
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
		updatePost(gitName)
	}
}

func getIndex(w http.ResponseWriter, r *http.Request) {
	content, err := os.ReadFile(dataDir + "_pages/index.md")

	if err != nil {
		getErrorPage(w, r, err.Error())
		return
	}

	htmlContent := toHTML(content, "_pages")

	recentPosts := publicPosts
	if len(recentPosts) > 5 {
		recentPosts = recentPosts[:5]
	}

	files, err := filepath.Glob(tmplPath)
	if err != nil {
		getErrorPage(w, r, err.Error())
		return
	}

	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		getErrorPage(w, r, err.Error())
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

	err = tmpl.ExecuteTemplate(w, "index.tmpl.html", data)
	if err != nil {
		getErrorPage(w, r, err.Error())
		return
	}
}

func getPage(w http.ResponseWriter, r *http.Request) {
	pageName := chi.URLParam(r, "pageName")
	if pageName == "posts" {
		getPosts(w, r)
		return
	}

	content, err := os.ReadFile(dataDir + "_pages/" + pageName + ".md")
	if err != nil {
		getErrorPage(w, r, err.Error())
		return
	}
	htmlContent := toHTML(content, "/posts/_pages")

	data := struct {
		Title        string
		Header       string
		MarkdownHTML template.HTML
	}{
		Title:        config.BlogTitle + " - " + pageName,
		Header:       config.BlogHeader,
		MarkdownHTML: template.HTML(htmlContent),
	}

	files, err := filepath.Glob(tmplPath)
	if err != nil {
		getErrorPage(w, r, err.Error())
		return
	}

	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		getErrorPage(w, r, err.Error())
		return
	}

	err = tmpl.ExecuteTemplate(w, "post.tmpl.html", data)
	if err != nil {
		getErrorPage(w, r, err.Error())
		return
	}
}

func getPosts(w http.ResponseWriter, r *http.Request) {
	files, err := filepath.Glob(tmplPath)
	if err != nil {
		getErrorPage(w, r, err.Error())
		return
	}

	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		getErrorPage(w, r, err.Error())
		return
	}

	data := struct {
		Title  string
		Header string
		Posts  []Post
	}{
		Title:  config.BlogTitle + " - posts",
		Header: config.BlogHeader,
		Posts:  publicPosts,
	}

	err = tmpl.ExecuteTemplate(w, "posts.tmpl.html", data)
	if err != nil {
		getErrorPage(w, r, err.Error())
		return
	}
}

func getPost(w http.ResponseWriter, r *http.Request) {
	postName := chi.URLParam(r, "postName")

	// check if the post is public
	postInfo, err := getPostInfo(postName)
	if err != nil {
		getErrorPage(w, r, "Post not found.")
		return
	}
	if postInfo.State != "public" {
		getErrorPage(w, r, "Post not found.")
		return
	}

	content, err := os.ReadFile(dataDir + postName + "/README.md")
	if err != nil {
		getErrorPage(w, r, err.Error())
		return
	}

	htmlContent := toHTML(content, postName)

	files, err := filepath.Glob(tmplPath)
	if err != nil {
		getErrorPage(w, r, err.Error())
		return
	}

	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		getErrorPage(w, r, err.Error())
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

	err = tmpl.ExecuteTemplate(w, "post.tmpl.html", data)
	if err != nil {
		getErrorPage(w, r, err.Error())
		return
	}
}

func getErrorPage(w http.ResponseWriter, r *http.Request, errMessage string) {
	files, err := filepath.Glob(tmplPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Title  string
		Header string
		Error  string
	}{
		Title:  config.BlogHeader + " - ERROR",
		Header: config.BlogHeader,
		Error:  errMessage,
	}

	err = tmpl.ExecuteTemplate(w, "error.tmpl.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func servePostAssets(w http.ResponseWriter, r *http.Request) {
	postName := chi.URLParam(r, "postName")

	// check if the post is public
	postInfo, err := getPostInfo(postName)
	if err != nil {
		getErrorPage(w, r, "Post not found.")
		return
	}
	if postInfo.State != "public" {
		getErrorPage(w, r, "Post not found.")
		return
	}

	fileName := strings.TrimPrefix(r.URL.Path, "/posts/"+postName+"/")
	if filepath.Ext(fileName) == ".md" {
		content, err := os.ReadFile(dataDir + postName + "/" + fileName)
		if err != nil {
			getErrorPage(w, r, err.Error())
			return
		}

		htmlContent := toHTML(content, "")

		files, err := filepath.Glob(tmplPath)
		if err != nil {
			getErrorPage(w, r, err.Error())
			return
		}

		tmpl, err := template.ParseFiles(files...)
		if err != nil {
			getErrorPage(w, r, err.Error())
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
			Title:        config.BlogHeader + " - " + post.Name,
			Header:       config.BlogHeader,
			MarkdownHTML: template.HTML(htmlContent),
		}

		err = tmpl.ExecuteTemplate(w, "post.tmpl.html", data)
		if err != nil {
			getErrorPage(w, r, err.Error())
			return
		}
	} else {
		http.StripPrefix("/posts/"+postName+"/", http.FileServer(http.Dir(dataDir+postName+"/"))).ServeHTTP(w, r)
	}
}

func toHTML(content []byte, postName string) []byte {
	contentStr := string(content)
	contentStr = strings.ReplaceAll(contentStr, "\\(", "$")
	contentStr = strings.ReplaceAll(contentStr, "\\)", "$")
	contentStr = strings.ReplaceAll(contentStr, "\\[", "$$")
	contentStr = strings.ReplaceAll(contentStr, "\\]", "$$")
	content = []byte(contentStr)

	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(content)

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	htmlContent := markdown.Render(doc, renderer)

	htmlContent = []byte(replacePaths(string(htmlContent), postName))
	replaced, err := replaceCodeParts(htmlContent)
	if err != nil {
		log.Println("unable to highlight code block")
		return htmlContent
	}
	htmlContent = []byte(replaced)

	return htmlContent
}

func replacePaths(htmlContent string, dirPath string) string {
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

	// find href links
	linkTagStart := "<a href="
	linkTags := strings.Split(htmlContent, linkTagStart)

	for i := 1; i < len(linkTags); i++ {
		linkTag := linkTagStart + linkTags[i]

		// Find the src attribute within the <img> tag
		hrefStartIndex := strings.Index(linkTag, "\"")
		if hrefStartIndex == -1 {
			continue
		}
		hrefEndIndex := strings.Index(linkTag[hrefStartIndex+1:], "\"")
		if hrefEndIndex == -1 {
			continue
		}

		// Extract the image filename from the src attribute
		href := linkTag[hrefStartIndex+1 : hrefStartIndex+1+hrefEndIndex]

		// If the src is a url, skip it
		if strings.HasPrefix(href, "http") {
			continue
		}

		// Build the correct relative path to the image
		relLinkPath := filepath.Join(dirPath, href) + ` target="_blank"`

		// Update the src attribute with the correct URL
		updatedLinkTag := strings.Replace(linkTag, href, relLinkPath, 1)

		// Replace the original <img> tag with the updated one
		htmlContent = strings.Replace(htmlContent, linkTagStart+linkTags[i], updatedLinkTag, 1)
	}

	return htmlContent
}

func replaceCodeParts(mdFile []byte) (string, error) {
	byteReader := bytes.NewReader(mdFile)
	doc, err := goquery.NewDocumentFromReader(byteReader)
	if err != nil {
		return "", err
	}
	// find code-parts via css selector and replace them with highlighted versions
	doc.Find("code[class*=\"language-\"]").Each(func(i int, s *goquery.Selection) {
		oldCode := s.Text()
		formatted, err := syntaxhighlight.AsHTML([]byte(oldCode))
		if err != nil {
			log.Fatal(err)
		}
		s.SetHtml(string(formatted))
	})
	new, err := doc.Html()
	if err != nil {
		return "", err
	}
	// replace unnecessarily added html tags
	new = strings.Replace(new, "<html><head></head><body>", "", 1)
	new = strings.Replace(new, "</body></html>", "", 1)
	return new, nil
}
