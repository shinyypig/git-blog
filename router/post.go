package router

import (
	"encoding/json"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/russross/blackfriday/v2"
)

const postListPath = dataDir + "_pages/postsList.json"

type Post struct {
	Name   string
	Title  string
	Body   template.HTML
	Banner string
	Mtime  string
	State  string
	Hash   string
}

var posts []Post
var publicPosts []Post

func checkAllPosts() error {
	files, err := os.ReadDir(dataDir)
	if err != nil {
		return err
	}

	posts = []Post{}
	for _, file := range files {
		if file.IsDir() {
			post, err := getPostInfo(file.Name())
			if err != nil {
				continue
			}
			posts = append(posts, post)
		}
	}
	sortPosts()
	getPublicPosts()
	savePosts()

	return nil
}

func sortPosts() {
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Mtime > posts[j].Mtime
	})
}

func getPublicPosts() {
	publicPosts = []Post{}
	for _, post := range posts {
		if post.State == "public" {
			publicPosts = append(publicPosts, post)
		}
	}
}

func getPostInfo(name string) (Post, error) {
	post := Post{}

	_, err := os.Stat(dataDir + name)
	if err != nil {
		return post, err
	}

	mdContent, err := os.ReadFile(dataDir + name + "/README.md")
	if err != nil || name == "_pages" || name == "_config" {
		post = Post{
			Name:   name,
			Title:  "",
			Body:   "",
			Banner: "",
			Mtime:  getLatestCommitDate(repoDir + name),
			State:  "private",
			Hash:   getLastestCommitHash(repoDir + name),
		}
		return post, nil
	}

	// extract the first line
	firstLine := strings.Split(string(mdContent), "\n")[0]
	state := config.PostDefaultState
	if strings.HasPrefix(firstLine, "<!--") {
		if strings.Contains(firstLine, "public") {
			state = "public"
		} else if strings.Contains(firstLine, "private") {
			state = "private"
		} else if strings.Contains(firstLine, "delete") {
			state = "delete"
		}
	}

	htmlContent := blackfriday.Run(mdContent)
	title, body, banner := extractTitleAndBody(htmlContent)
	print("title:", title)
	print("body:", body)
	print("banner:", banner)
	post = Post{
		Name:   name,
		Title:  title,
		Body:   body,
		Banner: dataDir + name + "/" + banner,
		Mtime:  getLatestCommitDate(repoDir + name),
		State:  state,
		Hash:   getLastestCommitHash(repoDir + name),
	}
	return post, nil
}

func updatePost(name string) []Post {
	post, err := getPostInfo(name)
	if err != nil || post.State == "delete" {
		posts = deletePost(name)
	} else {
		flag := false
		for i, p := range posts {
			if p.Name == name {
				posts[i] = post
				flag = true
				break
			}
		}
		if !flag {
			posts = append(posts, post)
		}
	}

	sortPosts()
	getPublicPosts()
	savePosts()

	return posts
}

func deletePost(name string) []Post {
	for i, post := range posts {
		if post.Name == name {
			posts = append(posts[:i], posts[i+1:]...)
			break
		}
	}
	os.RemoveAll(dataDir + name)
	os.RemoveAll(repoDir + name)
	return posts
}

func getLatestCommitDate(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%cd", "--date=format:%Y-%m-%d %H:%M:%S")
	output, err := cmd.Output()
	if err != nil {
		return time.Now().Format("2006-01-02 15:04:05")
	}
	return strings.TrimSpace(string(output))
}

func getLastestCommitHash(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "main")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func savePosts() {
	jsonData, _ := json.MarshalIndent(posts, "", "  ")
	os.WriteFile(postListPath, jsonData, 0644)
	// git add postList.json and commit and push to local repo
	cmd := exec.Command("git", "-C", dataDir+"_pages", "add", "postsList.json")
	cmd.Run()
	cmd = exec.Command("git", "-C", dataDir+"_pages", "commit", "-m", "update postList.json")
	cmd.Run()
	cmd = exec.Command("git", "-C", dataDir+"_pages", "push", "origin", "main")
	cmd.Run()
	log.Println("postList.json saved")
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
	title = strings.Replace(title, "&ldquo;", "\"", -1)
	title = strings.Replace(title, "&rdquo;", "\"", -1)
	title = strings.Replace(title, "&lsquo;", "'", -1)
	title = strings.Replace(title, "&rsquo;", "'", -1)
	title = strings.Replace(title, "&amp;", "&", -1)
	title = strings.Replace(title, "&gt;", ">", -1)
	title = strings.Replace(title, "&lt;", "<", -1)
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
	if len(body) >= 280 {
		for i := 260; i <= 280; i++ {
			if body[i] == ' ' {
				body = body[:i] + " ..."
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

func extractGitData(name string) {
	targetDir := dataDir + name

	// Check if the directory exists
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		// Directory exists, remove it
		err = os.RemoveAll(targetDir)
		if err != nil {
			log.Fatalf("Failed to remove existing directory: %v", err)
		}
	}

	// set HEAD to main branch
	cmd := exec.Command("git", "-C", repoDir+name, "symbolic-ref", "HEAD", "refs/heads/main")
	cmd.Run()

	// Clone the repo
	cmd = exec.Command("git", "clone", repoDir+name, targetDir)
	cmd.Run()

	// add local repo for _pages and remove the .git directory except the _pages folder
	if name != "_pages" {
		gitDir := filepath.Join(targetDir, ".git")
		err := os.RemoveAll(gitDir)
		if err != nil {
			log.Fatalf("Failed to remove .git directory: %v", err)
		}
	}
}

func extractAllGitData() {
	// Get all the directories in the data directory
	files, err := os.ReadDir(repoDir)
	if err != nil {
		log.Fatalf("Failed to read data directory: %v", err)
	}

	// Extract git data for each directory
	for _, file := range files {
		if file.IsDir() {
			extractGitData(file.Name())
		}
	}
}

func getPostsFromJson() {
	// read the posts list from json file
	postList, err := os.ReadFile(postListPath)
	if err != nil {
		return
	}
	// convert the json to []Post
	err = json.Unmarshal(postList, &posts)
	if err != nil {
		return
	}

	getPublicPosts()
}

func AnaylzePosts() {
	if config.AnaylzePostsOnStart {
		extractAllGitData()
		checkAllPosts()
		log.Println("All posts checked")
	} else {
		checkAllPosts()
		log.Println("Skip anaylzing posts")
	}
}
