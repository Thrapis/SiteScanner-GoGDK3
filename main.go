package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var (
	win         *gtk.Window
	entry       *gtk.Entry
	urlTreeView *gtk.TreeView
	treeStore   *gtk.TreeStore
	progressBar *gtk.ProgressBar
	button1     *gtk.Button
)

func standartErrorHandle(err error) {
	if err != nil {
		log.Fatal("Ошибка:", err)
	}
}

func setupWindow(win *gtk.Window) {
	win.SetIconFromFile("icon.ico")
	win.SetTitle("Site Scanner")
	win.Connect("destroy", func() {
		gtk.MainQuit()
	})
	win.SetPosition(gtk.WIN_POS_CENTER)
	win.SetDefaultSize(600, 300)
}

func startScanningProcess() {
	root := NewUrlTreeStruct("belstu.by")
	news := NewUrlTreeStruct("belstu.by/news")
	fakultety := NewUrlTreeStruct("belstu.by/fakultety")
	tov := NewUrlTreeStruct("belstu.by/fakultety/tov")
	htit := NewUrlTreeStruct("belstu.by/fakultety/htit")
	root.AppendChild(news)
	root.AppendChild(fakultety)
	fakultety.AppendChild(tov)
	fakultety.AppendChild(htit)

	text, err := entry.GetText()

	if err == nil {
		go func(url string) {
			glib.IdleAdd(func() {
				button1.SetSensitive(false)
				progressBar.SetShowText(true)
				progressBar.SetText("Process")
				progressBar.SetFraction(0)

			})
			StartScan(url, func(p float64) {
				glib.IdleAdd(func() {
					progressBar.SetFraction(p)
				})
			})
			glib.IdleAdd(func() {
				progressBar.SetFraction(1)
				progressBar.SetText(fmt.Sprintf("Process done at %s", time.Now().Format("15:04:05 02.01.2006")))
				applyTree(treeStore, root)
				button1.SetSensitive(true)
			})
		}(text)
	}
}

func main() {
	fmt.Println("start-----------------")
	gtk.Init(nil)

	question_pixbuf = getPixbuf("images/question.png")
	check_pixbuf = getPixbuf("images/check.png")
	remove_pixbuf = getPixbuf("images/remove.png")

	b, err := gtk.BuilderNew()
	standartErrorHandle(err)

	err = b.AddFromFile("ui/main_window.glade")
	standartErrorHandle(err)

	obj, err := b.GetObject("MainWindow")
	standartErrorHandle(err)

	win = obj.(*gtk.Window)
	setupWindow(win)

	obj, err = b.GetObject("StartUrlField")
	standartErrorHandle(err)
	entry = obj.(*gtk.Entry)
	entry.Connect("activate", func() {
		startScanningProcess()
	})

	obj, err = b.GetObject("UrlTreeView")
	standartErrorHandle(err)
	urlTreeView = obj.(*gtk.TreeView)
	treeStore = setupTreeView(urlTreeView)

	obj, err = b.GetObject("ProcessProgressBar")
	standartErrorHandle(err)
	progressBar = obj.(*gtk.ProgressBar)

	obj, err = b.GetObject("StartProcessButton")
	standartErrorHandle(err)

	button1 = obj.(*gtk.Button)
	button1.Connect("clicked", func() {
		startScanningProcess()
	})

	win.ShowAll()
	gtk.Main()
}
