package main

import (
	"fmt"
	"github.com/deckarep/gosx-notifier"
	"github.com/fsnotify/fsnotify"
	"github.com/gen2brain/beeep"
	"github.com/spf13/viper"
	"github.com/studio-b12/gowebdav"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	watcher    *fsnotify.Watcher
	excludes   = make([]string, 0)
	dav        *gowebdav.Client
	path       = ""
	server     = ""
	serverPath = ""
	user       = ""
	password   = ""
)

func main() {
	viper.SetConfigName("webdav") // name of config file (without extension)
	viper.SetConfigType("json")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	err := viper.ReadInConfig()   // Find and read the config file
	if err != nil {               // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}
	path = viper.GetString("local_path")
	server = viper.GetString("server")
	serverPath = viper.GetString("server_path")
	user = viper.GetString("username")
	password = viper.GetString("password")
	excludes = viper.GetStringSlice("ignores")

	root := strings.TrimSuffix(path, "/") + "/"
	dav = gowebdav.NewClient(server, user, password)
	u, err := url.Parse(server)
	if err != nil {
		fmt.Printf("Parse server failed with error: %s \n", err)
		return
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/"
	u.Path = strings.TrimPrefix(u.Path, "/")
	// creates a new file watcher
	watcher, _ = fsnotify.NewWatcher()
	defer func(watcher *fsnotify.Watcher) {
		err = watcher.Close()
		if err != nil {

		}
	}(watcher)

	// starting at the root of the project, walk each file/directory searching for
	// directories
	if err = filepath.Walk(path, watchDir); err != nil {
		fmt.Printf("Start watch failed with error: %s \n", err)
		return
	}

	done := make(chan bool)

	//
	go func() {
		for {
			select {
			// watch for events
			case event := <-watcher.Events:
				webdavFilePath := serverPath + "/" + strings.ReplaceAll(event.Name, root, "")
				webdavFilePath = strings.ReplaceAll(webdavFilePath, "//", "/")
				processEvent(event, webdavFilePath, root)
				// watch for errors
			case err = <-watcher.Errors:
				fmt.Printf("Watcher error %s \n", err)
			}
		}
	}()

	<-done
}

// watchDir gets run as a walk func, searching for directories to add watchers to
func watchDir(path string, fi os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	// since fsnotify can watch all the files in a directory, watchers only need
	// to be added to each nested directory
	if fi.Mode().IsDir() {
		found := false
		for _, exPath := range excludes {
			if strings.Contains(path, exPath) {
				found = true
			}
		}
		if !found {
			return watcher.Add(path)
		}
	}

	return nil
}

func processEvent(event fsnotify.Event, webdavFilePath, root string) {
	if (event.Name + "/") == root {
		log.Printf("root change ignored, event: %s, root: %s", event.Name, root)
		return
	}
	if strings.HasSuffix(event.Name, "~") {
		return
	}
	if event.Op == fsnotify.Chmod {
		return
	}
	var err error
	var title string
	if event.Op&fsnotify.Write == fsnotify.Write {
		file, _ := os.Open(event.Name)
		err = dav.WriteStream(webdavFilePath, file, 0644)
		_ = file.Close()
		title = "Upload success"
	}
	if event.Op&fsnotify.Create == fsnotify.Create {
		isDirectory := isDir(event.Name)
		_ = watcher.Add(event.Name)
		if isDirectory {
			err = dav.MkdirAll(webdavFilePath, 0755)
		} else {
			file, _ := os.Open(event.Name)
			err = dav.WriteStream(webdavFilePath, file, 0644)
			_ = file.Close()
		}
		title = "Upload success"
	}
	if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
		err = dav.Remove(webdavFilePath)
		_ = watcher.Remove(event.Name)
		title = "Remove success"
	}
	if err != nil {
		notify("Webdav Error", err.Error(), true)
		log.Printf("%s %s %s %s \n", event.Op.String(), webdavFilePath, "webdav error:", err.Error())
	} else {
		log.Printf("%s %s %s \n", event.Op.String(), webdavFilePath, strings.ToLower(title))
		notify(title, webdavFilePath, false)
	}
}

func isDir(path string) bool {
	i, err := os.Stat(path)
	if err != nil {
		return false
	}
	return i.IsDir()
}

func notifyMac(title, subtitle string, isAlert bool) {
	//At a minimum specifiy a message to display to end-user.
	note := gosxnotifier.NewNotification("Webdav sync notification")

	//Optionally, set a title
	note.Title = title

	//Optionally, set a subtitle
	note.Subtitle = subtitle

	//Optionally, set a sound from a predefined set.
	if isAlert {
		note.Sound = gosxnotifier.Default
	}

	//Optionally, set a group which ensures only one notification is ever shown replacing previous notification of same group id.
	note.Group = "com.webdav.sync.notifications"

	//Optionally, set a sender (Notification will now use the Safari icon)
	//note.Sender = "com.apple.Safari"

	//Optionally, specifiy a url or bundleid to open should the notification be
	//clicked.
	//note.Link = "http://www.yahoo.com" //or BundleID like: com.apple.Terminal

	//Optionally, an app icon (10.9+ ONLY)
	//note.AppIcon = "gopher.png"

	//Optionally, a content image (10.9+ ONLY)
	//note.ContentImage = "gopher.png"

	//Then, push the notification
	_ = note.Push()
}

func notify(title, subtitle string, isAlert bool) {
	systemOs := runtime.GOOS
	switch systemOs {
	case "darwin":
		notifyMac(title, subtitle, isAlert)
	default:
		if isAlert {
			_ = beeep.Alert(title, subtitle, "assets/warning.png")
		} else {
			_ = beeep.Notify(title, subtitle, "assets/information.png")
		}
	}
}
