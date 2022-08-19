package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var (
	win              *gtk.Window
	entry            *gtk.Entry
	urlTreeView      *gtk.TreeView
	treeStore        *gtk.TreeStore
	progressBar      *gtk.ProgressBar
	button1          *gtk.Button
	button2          *gtk.Button
	selectedUrlLink  *gtk.LinkButton
	innerUrlTreeView *gtk.TreeView
	listStore        *gtk.ListStore

	searchedUrl string
	listOfUrls  *[]string
	urlTree     *UrlTreeStruct
)

const (
	STATUS_NO_INFO = iota
	STATUS_PROBLEM
	STATUS_SUCCESS
	STATUS_LONGWAIT
	STATUS_TEAPOT
	STATUS_NOTFOUND
	STATUS_ROBOT
	STATUS_NOTALLOWED
	STATUS_FAILURE
)

const (
	max_pool       = 100
	max_outer_pool = 5
	max_inner_pool = 5
)

func standartErrorHandle(err error) {
	if err != nil {
		log.Fatal("Ошибка:", err)
	}
}

var progressMtx sync.Mutex

func progressChange(text string, progress float64) {
	glib.IdleAdd(func() {
		if progressMtx.TryLock() {
			progressBar.SetFraction(progress)
			progressBar.SetText(text)
			progressMtx.Unlock()
		}
	})
}

func progressChangeWithToolTip(text string, progress float64) {
	if progressMtx.TryLock() {
		progressBar.SetFraction(progress)
		progressBar.SetText(text)
		progressBar.SetTooltipText(text)
		progressMtx.Unlock()
	}
}

func setupWindow(win *gtk.Window) {
	win.SetIconFromFile("icon.ico")
	win.SetTitle("Site Scanner")
	win.Connect("destroy", func() {
		gtk.MainQuit()
	})
	win.SetPosition(gtk.WIN_POS_CENTER)
	win.SetDefaultSize(900, 600)
}

func lockUI() {
	glib.IdleAdd(func() {
		entry.SetSensitive(false)
		button1.SetSensitive(false)
		button2.SetSensitive(false)
	})
}

func unlockUI() {
	glib.IdleAdd(func() {
		entry.SetSensitive(true)
		button1.SetSensitive(true)
		button2.SetSensitive(true)
	})
}

func clearSelection() {
	selectedUrlLink.SetUri("")
	selectedUrlLink.SetLabel("")
	selectedUrlLink.SetSensitive(false)
	listStore.Clear()
}

func urlTreeSelectionChanged(s *gtk.TreeSelection) {
	//fmt.Println("Selection!")
	rows := s.GetSelectedRows(treeStore)
	item := rows.First()

	if item != nil {
		path := item.Data().(*gtk.TreePath)
		iter, _ := treeStore.GetIter(path)

		uts := urlTree.FindByTreeIter(iter)
		if uts != nil {
			selectedUrlLink.SetUri(uts.Url)
			selectedUrlLink.SetLabel(uts.Url)
			selectedUrlLink.SetSensitive(true)
			applyList(listStore, &uts.InnerUrls)
		} else {
			fmt.Println("No uts!")
		}
	} else {
		fmt.Println("No item!")
	}
}

func startScanningProcess() {
	clearSelection()
	text, err := entry.GetText()
	if err == nil {
		searchedUrl, err = Normalize(text)
		if err != nil {
			log.Panic("incorrect url:", err)
			return
		}
		go func(norm_url string) {
			time1 := time.Now()
			lockUI()
			message := "Process"
			progressChangeWithToolTip(message, 0)
			listOfUrls = StartScan(norm_url, progressChange)
			urlTree = NewUrlTreeStruct(norm_url)
			for _, page := range *listOfUrls {
				fmt.Printf("Href: %s\n", page)
				urlTree.AppendAccordingUrl(NewUrlTreeStruct(page))
			}
			time2 := time.Now()
			message = fmt.Sprintf("Process done at %s [%s]", time.Now().Format("15:04:05"), (time2.Sub(time1)))
			progressChangeWithToolTip(message, 1)
			glib.IdleAdd(func() {
				applyTree(treeStore, urlTree)
			})
			unlockUI()
		}(searchedUrl)
	}
}

func startCheckPages() {
	clearSelection()
	go func() {
		lockUI()
		message := "Process"
		progressChangeWithToolTip(message, 0)
		time1 := time.Now()
		CheckUrls(searchedUrl, listOfUrls, urlTree, progressChange)
		time2 := time.Now()
		message = fmt.Sprintf("Process done at %s [%s]", time.Now().Format("15:04:05"), (time2.Sub(time1)))
		progressChangeWithToolTip(message, 1)
		glib.IdleAdd(func() {
			applyTree(treeStore, urlTree)
		})
		unlockUI()
	}()
}

func main() {
	fmt.Println("start-----------------")
	gtk.Init(nil)

	clear_pixbuf = getPixbuf("images/clear.png")
	question_pixbuf = getPixbuf("images/question.png")
	check_pixbuf = getPixbuf("images/check.png")
	remove_pixbuf = getPixbuf("images/remove.png")
	wait_pixbuf = getPixbuf("images/wait.png")
	teapot_pixbuf = getPixbuf("images/teapot.png")
	not_found_pixbuf = getPixbuf("images/not_found.png")
	robot_pixbuf = getPixbuf("images/robot.png")
	not_allowed_pixbuf = getPixbuf("images/not_allowed.png")

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
	treeStore = setupTreeViewLikeTree(urlTreeView)
	sel, err := urlTreeView.GetSelection()
	standartErrorHandle(err)
	sel.Connect("changed", urlTreeSelectionChanged)

	obj, err = b.GetObject("ProcessProgressBar")
	standartErrorHandle(err)
	progressBar = obj.(*gtk.ProgressBar)

	obj, err = b.GetObject("StartProcessButton")
	standartErrorHandle(err)
	button1 = obj.(*gtk.Button)
	button1.Connect("clicked", func() {
		startScanningProcess()
	})

	obj, err = b.GetObject("CheckPagesButton")
	standartErrorHandle(err)
	button2 = obj.(*gtk.Button)
	button2.Connect("clicked", func() {
		startCheckPages()
	})

	obj, err = b.GetObject("InnerUrlTreeView")
	standartErrorHandle(err)
	innerUrlTreeView = obj.(*gtk.TreeView)
	listStore = setupTreeViewLikeList(innerUrlTreeView)

	obj, err = b.GetObject("SelectedUrlLink")
	standartErrorHandle(err)
	selectedUrlLink = obj.(*gtk.LinkButton)
	clearSelection()

	win.ShowAll()
	gtk.Main()
}
