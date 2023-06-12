package router

import (
	"encoding/json"
	"html/template"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/russross/blackfriday/v2"
)

const postListJson = dataDir + ".pages/postsList.json"

type Dir struct {
	Name  string
	Mtime time.Time
}

type Post struct {
	Path   string
	Title  string
	Body   template.HTML
	Src    string
	Mtime  string
	Public bool
}

func UpdatePosts(name string, action string) {
	var posts []Post
	postsJson, _ := os.ReadFile(postListJson)
	json.Unmarshal(postsJson, &posts)

	for i, post := range posts {
		if post.Path == name {
			if action == "publish" {
				posts[i].Public = true
			} else if action == "unpublish" {
				posts[i].Public = false
			}
			break
		}
	}

	postsJson, _ = json.Marshal(posts)
	os.WriteFile(postListJson, postsJson, 0644)
}

func CheckAllPosts() error {
	var dirs []Dir
	files, err := os.ReadDir(dataDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			if strings.HasPrefix(file.Name(), ".") {
				continue
			}
			readmePath := dataDir + file.Name() + "/README.md"
			if _, err := os.Stat(readmePath); os.IsNotExist(err) {
				continue
			}
			dirs = append(dirs, Dir{
				Name:  file.Name(),
				Mtime: getFileModifiedTime(dataDir + file.Name()),
			})
		}
	}

	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Mtime.After(dirs[j].Mtime)
	})

	posts := make([]Post, len(dirs))
	for i, dir := range dirs {

	}

	jsonData, err := json.Marshal(posts)

	if err != nil {
		return err
	}

	err = os.WriteFile(postListJson, jsonData, 0644)

	if err != nil {
		return err
	}

	return nil
}

func getPostInfo(name string) (Post, error) {
	post := Post{}
	mdContent, err := os.ReadFile(dataDir + name + "/README.md")
	if err != nil {
		return post, err
	}

	// extract the first line
	firstLine := strings.Split(string(mdContent), "\n")[0]
	ifPublic := false
	if strings.HasPrefix(firstLine, "<!--") {
		if strings.Contains(firstLine, "public") {
			ifPublic = true
		} else if strings.Contains(firstLine, "delete") {
			os.RemoveAll(dataDir + dir.Name)
		}
	}

	htmlContent := blackfriday.Run(mdContent)
	title, body, src := extractTitleAndBody(htmlContent)
	posts[i] = Post{
		Path:   "posts/" + dir.Name,
		Title:  title,
		Body:   body,
		Src:    dataDir + dir.Name + "/" + src,
		Mtime:  dir.Mtime.Format("2006-01-02 15:04:05"),
		Public: ifPublic,
	}
}

func getFileModifiedTime(filePath string) time.Time {
	info, err := os.Stat(filePath)
	if err != nil {
		return time.Now()
	}
	mtime := info.ModTime()

	return mtime
}

func extractTitleAndBody(html []byte) (string, template.HTML, string) {
	// 将HTML转换为字符串
	htmlContent := string(html)

	// 提取H1标题
	title := extractH1Title(htmlContent)

	// 提取第一段内容
	body := extractFirstParagraph(htmlContent)

	// extract src of the first image
	src := extractFirstImage(htmlContent)

	return title, body, src
}

func extractH1Title(htmlContent string) string {
	title := ""
	index := strings.Index(htmlContent, "<h1>")
	if index != -1 {
		endIndex := strings.Index(htmlContent[index:], "</h1>")
		if endIndex != -1 {
			title = strings.TrimSpace(htmlContent[index+4 : index+endIndex])
		}
	}
	return title
}

func extractFirstParagraph(htmlContent string) template.HTML {
	body := ""
	index := 0
	for paragraphIndex := strings.Index(htmlContent[index:], "<p>"); paragraphIndex != -1; paragraphIndex = strings.Index(htmlContent[index:], "<p>") {
		paragraphEndIndex := strings.Index(htmlContent[index+paragraphIndex:], "</p>")
		if paragraphEndIndex != -1 {
			body = strings.TrimSpace(htmlContent[index+paragraphIndex : index+paragraphIndex+paragraphEndIndex+4])
			// if image found, skip it and extract the next paragraph
			if strings.Contains(body, "<img") {
				index = index + paragraphEndIndex + 4
			} else {
				break
			}
		}
	}
	return template.HTML(body)
}

func extractFirstImage(htmlContent string) string {
	imgTagStart := "<img"
	imgTags := strings.Split(htmlContent, imgTagStart)

	// if no image found, return empty string
	if len(imgTags) < 2 {
		return ""
	} else {
		imgTagEnd := strings.Index(imgTags[1], ">")
		if imgTagEnd != -1 {
			imgTag := imgTags[1][:imgTagEnd]
			srcIndex := strings.Index(imgTag, "src=\"")
			if srcIndex != -1 {
				imgSrc := imgTag[srcIndex+5:]
				imgSrcEnd := strings.Index(imgSrc, "\"")
				if imgSrcEnd != -1 {
					return imgSrc[:imgSrcEnd]
				}
			}
		}
	}
	return ""
}
