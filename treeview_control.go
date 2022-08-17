package main

import (
	"log"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

const (
	COLUMN_IMG = iota
	COLUMN_TEXT
)

var (
	question_pixbuf *gdk.Pixbuf
	check_pixbuf    *gdk.Pixbuf
	remove_pixbuf   *gdk.Pixbuf
)

func getPixbuf(path string) *gdk.Pixbuf {
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

func setupTreeView(treeView *gtk.TreeView) *gtk.TreeStore {
	treeView.AppendColumn(createImageColumn("Status", COLUMN_IMG))
	treeView.AppendColumn(createTextColumn("Url", COLUMN_TEXT))

	treeStore, err := gtk.TreeStoreNew(gdk.PixbufGetType(), glib.TYPE_STRING)
	if err != nil {
		log.Fatal("Unable to create list store:", err)
	}

	treeView.SetModel(treeStore)

	return treeStore
}

func applyTree(store *gtk.TreeStore, root *UrlTreeStruct) {
	store.Clear()
	applyTreeBranch(store, nil, root)
}

func applyTreeBranch(store *gtk.TreeStore, parentIter *gtk.TreeIter, child *UrlTreeStruct) {

	iter := store.Append(parentIter)

	err := treeStore.SetValue(iter, COLUMN_IMG, question_pixbuf)
	if err != nil {
		log.Fatal("Unable config row:", err)
	}

	err = treeStore.SetValue(iter, COLUMN_TEXT, child.GetUrlAccordingParent())
	if err != nil {
		log.Fatal("Unable config row:", err)
	}

	for _, chld := range child.Childs {
		applyTreeBranch(store, iter, chld)
	}
}
