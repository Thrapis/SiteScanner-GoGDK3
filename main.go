package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var (
	win                   *gtk.Window
	entry                 *gtk.Entry
	urlTreeView           *gtk.TreeView
	treeStore             *gtk.TreeStore
	progressBar           *gtk.ProgressBar
	searchButton          *gtk.Button
	checkAllButton        *gtk.Button
	checkSingleButton     *gtk.Button
	settingsButton        *gtk.Button
	checkSingleDeepButton *gtk.Button
	saveButton            *gtk.Button
	loadButton            *gtk.Button
	selectedUrlLink       *gtk.LinkButton
	innerUrlTreeView      *gtk.TreeView
	listStore             *gtk.ListStore

	searchedUrl   string
	listOfUrls    *[]string
	listOfUrlTree []UrlTreeStruct
	urlTree       *UrlTreeStruct
	selectedUrl   *UrlTreeStruct
)

const (
	STATUS_NO_INFO = iota
	STATUS_PROBLEM
	STATUS_LONGWAIT
	STATUS_FAILURE
	STATUS_SUCCESS    = 200
	STATUS_NOTFOUND   = 404
	STATUS_NOTALLOWED = 405
	STATUS_TEAPOT     = 418
	STATUS_TMR        = 429
	STATUS_ISE        = 500
	STATUS_ROBOT      = 999
)

const (
	max_pool       = 200
	max_outer_pool = 10
	max_inner_pool = 25
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
	glib.IdleAdd(func() {
		progressMtx.Lock()
		progressBar.SetFraction(progress)
		progressBar.SetText(text)
		progressBar.SetTooltipText(text)
		progressMtx.Unlock()
	})
}

func setupWindow() {
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
		searchButton.SetSensitive(false)
		checkAllButton.SetSensitive(false)
		checkSingleButton.SetSensitive(false)
		settingsButton.SetSensitive(false)
		checkSingleDeepButton.SetSensitive(false)
		saveButton.SetSensitive(false)
		loadButton.SetSensitive(false)
	})
}

func unlockUI() {
	glib.IdleAdd(func() {
		entry.SetSensitive(true)
		searchButton.SetSensitive(true)
		checkAllButton.SetSensitive(true)
		checkSingleButton.SetSensitive(true)
		settingsButton.SetSensitive(true)
		checkSingleDeepButton.SetSensitive(true)
		saveButton.SetSensitive(true)
		loadButton.SetSensitive(true)
	})
}

func saveTo(path string) {
	bts, _ := json.Marshal(listOfUrlTree)
	file, _ := os.OpenFile(path, os.O_CREATE|os.O_TRUNC, 0777)
	file.Write(bts)
	file.Close()
}

func loadFrom(path string) {
	bts, _ := os.ReadFile(path)
	listOfUrlTree := make([]UrlTreeStruct, 0)
	err := json.Unmarshal(bts, listOfUrlTree)
	standartErrorHandle(err)
}

func clearSelection() {
	selectedUrlLink.SetUri("")
	selectedUrlLink.SetLabel("")
	selectedUrlLink.SetSensitive(false)
	selectedUrl = nil
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
			fmt.Println("Selectiong", uts.Url)
			selectedUrl = uts
			selectedUrlLink.SetUri(uts.Url)
			if len(uts.Url) > 48 {
				short := uts.Url[0:45] + "..."
				selectedUrlLink.SetLabel(short)
			} else {
				selectedUrlLink.SetLabel(uts.Url)
			}
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
			listOfUrlTree = append(listOfUrlTree, *urlTree)
			for _, page := range *listOfUrls {
				fmt.Printf("Href: %s\n", page)
				nts := NewUrlTreeStruct(page)
				listOfUrlTree = append(listOfUrlTree, *nts)
				urlTree.AppendAccordingUrl(nts)
			}
			time2 := time.Now()
			message = fmt.Sprintf("Process done at %s [%s]", time.Now().Format("15:04:05"), (time2.Sub(time1)))
			progressChangeWithToolTip(message, 1)
			glib.IdleAdd(func() {
				applyTree(treeStore, urlTree)
			})
			saveTo("card.json")
			unlockUI()
		}(searchedUrl)
	}
}

func startCheckPages() {
	go func() {
		lockUI()
		currentSelection := selectedUrl
		message := "Process"
		progressChangeWithToolTip(message, 0)
		time1 := time.Now()
		InitCheckUrls(searchedUrl, listOfUrls, urlTree, progressChange)
		time2 := time.Now()
		message = fmt.Sprintf("Process done at %s [%s]", time.Now().Format("15:04:05"), (time2.Sub(time1)))
		progressChangeWithToolTip(message, 1)
		glib.IdleAdd(func() {
			applyTree(treeStore, urlTree)
			fmt.Println(urlTree.FindByUrl(selectedUrl.Url))
			if currentSelection != nil {
				expandToItem(urlTreeView, treeStore, currentSelection)
			}
		})
		unlockUI()
	}()
}

func startCheckSelectedPage() {
	if selectedUrl == nil {
		return
	}
	go func() {
		lockUI()
		currentSelection := selectedUrl
		message := "Process"
		progressChangeWithToolTip(message, 0)
		time1 := time.Now()
		InitCheckUrl(searchedUrl, currentSelection, progressChange)
		time2 := time.Now()
		message = fmt.Sprintf("Process done at %s [%s]", time.Now().Format("15:04:05"), (time2.Sub(time1)))
		progressChangeWithToolTip(message, 1)
		glib.IdleAdd(func() {
			applyTree(treeStore, urlTree)
			if currentSelection != nil {
				expandToItem(urlTreeView, treeStore, currentSelection)
			}
		})
		unlockUI()
	}()
}

func startCheckPagesFromSelected() {

	go func() {
		lockUI()
		currentSelection := selectedUrl
		message := "Process"
		progressChangeWithToolTip(message, 0)
		time1 := time.Now()
		InitCheckUrlDeep(searchedUrl, currentSelection, progressChange)
		time2 := time.Now()
		message = fmt.Sprintf("Process done at %s [%s]", time.Now().Format("15:04:05"), (time2.Sub(time1)))
		progressChangeWithToolTip(message, 1)
		glib.IdleAdd(func() {
			applyTree(treeStore, urlTree)
		})
		unlockUI()
		if currentSelection != nil {
			expandToItem(urlTreeView, treeStore, currentSelection)
		}
	}()
}

func settings(m string) {
	dialog, _ := gtk.DialogNew()
	dialog.SetTitle(m)
	dialog.SetPosition(gtk.WIN_POS_CENTER)
	dialog.SetDefaultSize(100, 100)

	numEntry, _ := gtk.EntryNew()
	// numEntry.Connect("insert-text", func(ctx *glib.CallbackContext) {
	// 	a := (*[2000]uint8)(unsafe.Pointer(ctx.Args(0)))
	// 	p := (*int)(unsafe.Pointer(ctx.Args(2)))
	// 	i := 0
	// 	for a[i] != 0 {
	// 		i++
	// 	}
	// 	s := string(a[0:i])
	// 	if s == "." {
	// 		if *p == 0 {
	// 			input.StopEmission("insert-text")
	// 		}
	// 	} else {
	// 		_, err := strconv.Atof64(s)
	// 		if err != nil {
	// 			input.StopEmission("insert-text")
	// 		}
	// 	}
	// })

	dialog.Add(numEntry)

	dialog.Run()
	dialog.Destroy()
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
	tmr_pixbuf = getPixbuf("images/too_many_requests.png")
	ise_pixbuf = getPixbuf("images/internal_server_error.png")
	src_pixbuf = getPixbuf("images/img.png")
	href_pixbuf = getPixbuf("images/link.png")

	b, err := gtk.BuilderNew()
	standartErrorHandle(err)

	err = b.AddFromFile("ui/main_window.glade")
	standartErrorHandle(err)

	obj, err := b.GetObject("MainWindow")
	standartErrorHandle(err)

	win = obj.(*gtk.Window)
	setupWindow()

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
	searchButton = obj.(*gtk.Button)
	searchButton.Connect("clicked", func() {
		startScanningProcess()
	})

	obj, err = b.GetObject("CheckPagesButton")
	standartErrorHandle(err)
	checkAllButton = obj.(*gtk.Button)
	checkAllButton.Connect("clicked", func() {
		startCheckPages()
	})

	obj, err = b.GetObject("CheckSelectedPage")
	standartErrorHandle(err)
	checkSingleButton = obj.(*gtk.Button)
	checkSingleButton.Connect("clicked", func() {
		startCheckSelectedPage()
	})

	obj, err = b.GetObject("CheckPagesFromSelectedButton")
	standartErrorHandle(err)
	checkSingleDeepButton = obj.(*gtk.Button)
	checkSingleDeepButton.Connect("clicked", func() {
		startCheckPagesFromSelected()
	})

	obj, err = b.GetObject("SettingButton")
	standartErrorHandle(err)
	settingsButton = obj.(*gtk.Button)
	img, _ := gtk.ImageNewFromFile("images/gear.png")
	img.Show()
	settingsButton.SetImage(img)
	settingsButton.Connect("clicked", func() {
		settings("Some title")
	})

	obj, err = b.GetObject("SaveButton")
	standartErrorHandle(err)
	saveButton = obj.(*gtk.Button)
	saveButton.Connect("clicked", func() {
		saveTo("card.json")
	})

	obj, err = b.GetObject("LoadButton")
	standartErrorHandle(err)
	loadButton = obj.(*gtk.Button)
	loadButton.Connect("clicked", func() {
		loadFrom("card.json")
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
