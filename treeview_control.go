package main

import (
	"log"
	"sync"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

const (
	ONE_COLUMN_IMG = iota
	ONE_COLUMN_TEXT
)

const (
	TWO_COLUMN_IMG = iota
	TWO_COLUMN_IMG_2
	TWO_COLUMN_SIZE
	TWO_COLUMN_TEXT
)

var (
	clear_pixbuf       *gdk.Pixbuf
	question_pixbuf    *gdk.Pixbuf
	check_pixbuf       *gdk.Pixbuf
	remove_pixbuf      *gdk.Pixbuf
	wait_pixbuf        *gdk.Pixbuf
	teapot_pixbuf      *gdk.Pixbuf
	not_found_pixbuf   *gdk.Pixbuf
	robot_pixbuf       *gdk.Pixbuf
	not_allowed_pixbuf *gdk.Pixbuf
	tmr_pixbuf         *gdk.Pixbuf
	ise_pixbuf         *gdk.Pixbuf
	href_pixbuf        *gdk.Pixbuf
	src_pixbuf         *gdk.Pixbuf
)

var pixbufMtx sync.Mutex

func getPixbuf(path string) *gdk.Pixbuf {
	pixbufMtx.Lock()
	defer pixbufMtx.Unlock()
	img, err := gtk.ImageNewFromFile(path)
	if err != nil {
		log.Fatal("Unable to load pixbuf:", err)
	}
	return img.GetPixbuf()
}

func createImageColumn(title string, id int) *gtk.TreeViewColumn {
	cellRenderer, err := gtk.CellRendererPixbufNew()
	if err != nil {
		log.Fatal("Unable to create pixbuf cell renderer:", err)
	}
	column, err := gtk.TreeViewColumnNewWithAttribute(title, cellRenderer, "pixbuf", id)
	if err != nil {
		log.Fatal("Unable to create cell column:", err)
	}
	return column
}

func createTextColumn(title string, id int) *gtk.TreeViewColumn {
	cellRenderer, err := gtk.CellRendererTextNew()
	if err != nil {
		log.Fatal("Unable to create text cell renderer:", err)
	}
	column, err := gtk.TreeViewColumnNewWithAttribute(title, cellRenderer, "text", id)
	if err != nil {
		log.Fatal("Unable to create cell column:", err)
	}
	return column
}

func setupTreeViewLikeTree(treeView *gtk.TreeView) *gtk.TreeStore {
	treeView.AppendColumn(createImageColumn("Status", ONE_COLUMN_IMG))
	treeView.AppendColumn(createTextColumn("Url", ONE_COLUMN_TEXT))
	treeStore, err := gtk.TreeStoreNew(gdk.PixbufGetType(), glib.TYPE_STRING)
	if err != nil {
		log.Fatal("Unable to create list store:", err)
	}
	treeView.SetModel(treeStore)
	return treeStore
}

func setupTreeViewLikeList(treeView *gtk.TreeView) *gtk.ListStore {
	treeView.AppendColumn(createImageColumn("Intent", TWO_COLUMN_IMG))
	treeView.AppendColumn(createImageColumn("Status", TWO_COLUMN_IMG_2))
	treeView.AppendColumn(createTextColumn("Size", TWO_COLUMN_SIZE))
	treeView.AppendColumn(createTextColumn("Url", TWO_COLUMN_TEXT))
	treeStore, err := gtk.ListStoreNew(gdk.PixbufGetType(), gdk.PixbufGetType(), glib.TYPE_STRING, glib.TYPE_STRING)
	if err != nil {
		log.Fatal("Unable to create list store:", err)
	}
	treeView.SetModel(treeStore)
	return treeStore
}

func getPixbufByStatus(status int) *gdk.Pixbuf {
	switch status {
	case STATUS_NO_INFO:
		return clear_pixbuf
	case STATUS_PROBLEM:
		return question_pixbuf
	case STATUS_SUCCESS:
		return check_pixbuf
	case STATUS_FAILURE:
		return remove_pixbuf
	case STATUS_LONGWAIT:
		return wait_pixbuf
	case STATUS_TEAPOT:
		return teapot_pixbuf
	case STATUS_NOTFOUND:
		return not_found_pixbuf
	case STATUS_ROBOT:
		return robot_pixbuf
	case STATUS_TMR:
		return tmr_pixbuf
	case STATUS_NOTALLOWED:
		return not_allowed_pixbuf
	case STATUS_ISE:
		return ise_pixbuf
	default:
		return remove_pixbuf
	}
}

func getPixbufByIntent(intent int) *gdk.Pixbuf {
	switch intent {
	case INTENT_HREF:
		return href_pixbuf
	case INTENT_SRC:
		return src_pixbuf
	default:
		return clear_pixbuf
	}
}

func applyTree(store *gtk.TreeStore, root *UrlTreeStruct) {
	store.Clear()
	applyTreeBranch(store, nil, root)
}

func applyTreeBranch(store *gtk.TreeStore, parentIter *gtk.TreeIter, child *UrlTreeStruct) {
	iter := store.Append(parentIter)
	child.TreeIter = iter
	selected_pixbuf := getPixbufByStatus(child.Status)
	if selected_pixbuf != nil {
		err := treeStore.SetValue(iter, ONE_COLUMN_IMG, selected_pixbuf)
		if err != nil {
			log.Fatal("Unable config row:", err)
		}
	}
	err := treeStore.SetValue(iter, ONE_COLUMN_TEXT, child.GetUrlAccordingParent())
	if err != nil {
		log.Fatal("Unable config row:", err)
	}
	for _, chld := range child.Childs {
		applyTreeBranch(store, iter, chld)
	}
}

func applyList(store *gtk.ListStore, list *[]UrlStruct) {
	store.Clear()
	for _, us := range *list {
		intent_pixbuf := getPixbufByIntent(us.Intent)
		status_pixbuf := getPixbufByStatus(us.Status)
		store.Set(store.Append(), []int{TWO_COLUMN_IMG, TWO_COLUMN_IMG_2, TWO_COLUMN_SIZE, TWO_COLUMN_TEXT},
			[]interface{}{intent_pixbuf, status_pixbuf, us.GetShortSizeFormat(), us.Url})
	}
}

func expandToItem(treeView *gtk.TreeView, store *gtk.TreeStore, node *UrlTreeStruct) {
	path, _ := store.GetPath(node.TreeIter)
	treeView.ExpandToPath(path)
	selection, _ := treeView.GetSelection()
	selection.SelectIter(node.TreeIter)
	col := treeView.GetColumn(ONE_COLUMN_IMG)
	treeView.ScrollToCell(path, col, true, 0, 0)
}
