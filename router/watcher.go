package router

import (
	"fmt"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
)

type NotifyFile struct {
	watch *fsnotify.Watcher
}

func NewNotifyFile() *NotifyFile {
	w := new(NotifyFile)
	w.watch, _ = fsnotify.NewWatcher()
	return w
}

func (notifyFile *NotifyFile) WatchDir(dir string) {
	notifyFile.watch.Add(dir)
	files, _ := os.ReadDir(dir)
	for _, file := range files {
		if file.IsDir() {
			path := dir + "/" + file.Name()
			notifyFile.watch.Add(path)
		}
	}
	go notifyFile.WatchEvent()
}

func (notifyFile *NotifyFile) WatchEvent() {
	for {
		select {
		case ev := <-notifyFile.watch.Events:
			{
				if ev.Op&fsnotify.Create == fsnotify.Create {
					file, err := os.Stat(ev.Name)
					if err == nil && file.IsDir() {
						notifyFile.watch.Add(ev.Name)
						UpdatePosts()
					}
					if strings.HasSuffix(strings.ToLower(ev.Name), "read.md") {
						UpdatePosts()
					}
				}

				if ev.Op&fsnotify.Write == fsnotify.Write {
					fmt.Println("写入文件 : ", ev.Name)
				}

				if ev.Op&fsnotify.Remove == fsnotify.Remove {
					fi, err := os.Stat(ev.Name)
					if err == nil && fi.IsDir() {
						notifyFile.watch.Remove(ev.Name)
					}
				}

				if ev.Op&fsnotify.Rename == fsnotify.Rename {
					notifyFile.watch.Remove(ev.Name)
				}
			}
		case err := <-notifyFile.watch.Errors:
			{
				fmt.Println("error : ", err)
				return
			}
		}
	}
}
