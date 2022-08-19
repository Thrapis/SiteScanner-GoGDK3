package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gotk3/gotk3/gtk"
)

const (
	LINK_TYPE_PAGE = iota
	LINK_TYPE_FILE
	LINK_TYPE_MAILTO
	LINK_TYPE_TEL
	LINK_TYPE_CALLTO
)

type UrlStruct struct {
	Url        string
	Status     int
	LinkType   int
	SourceSize int64
}

func NewUrlStruct(url string) *UrlStruct {
	return &UrlStruct{Url: url, Status: STATUS_NO_INFO}
}

func (us UrlStruct) GetSizeB() float64 {
	return float64(us.SourceSize)
}

func (us UrlStruct) GetSizeKiB() float64 {
	return float64(us.SourceSize) / 1024
}

func (us UrlStruct) GetSizeMiB() float64 {
	return float64(us.SourceSize) / 1024 / 1024
}

func (us UrlStruct) GetShortSizeFormat() string {
	var size string
	if us.SourceSize == -1 {
		size = "Unknown"
	} else {
		size = fmt.Sprintf("%.2f MiB", us.GetSizeMiB())
		if size == "0.00 MiB" {
			size = fmt.Sprintf("%.2f KiB", us.GetSizeKiB())
		}
		if size == "0.00 KiB" {
			size = fmt.Sprintf("%.2f B", us.GetSizeB())
		}
	}
	return size
}

type UrlTreeStruct struct {
	Url        string
	Status     int
	Parent     *UrlTreeStruct
	Childs     []*UrlTreeStruct
	childMutex sync.Mutex
	InnerUrls  []*UrlStruct
	innerMutex sync.Mutex
	TreeIter   *gtk.TreeIter
}

func NewUrlTreeStruct(url string) *UrlTreeStruct {
	return &UrlTreeStruct{Url: url, Status: STATUS_NO_INFO, Childs: make([]*UrlTreeStruct, 0)}
}

func (uts *UrlTreeStruct) AppendInnerUrl(newInnerUrl *UrlStruct) {
	uts.innerMutex.Lock()
	uts.InnerUrls = append(uts.InnerUrls, newInnerUrl)
	uts.innerMutex.Unlock()
}

func (p *UrlTreeStruct) AppendChild(newChild *UrlTreeStruct) bool {
	if newChild == nil || newChild == p || p.Parent == newChild {
		return false
	}
	p.childMutex.Lock()
	newChild.Parent = p
	p.Childs = append(p.Childs, newChild)
	p.childMutex.Unlock()
	return true
}

func (uts *UrlTreeStruct) GetUrlAccordingParent() string {
	if uts.Parent == nil {
		return uts.Url
	}
	return strings.ReplaceAll(uts.Url, uts.Parent.Url, "")
}

func (r *UrlTreeStruct) FindByUrl(url string) *UrlTreeStruct {
	target := r
	for {
		if target.Url == url {
			return target
		}
		for i, uts := range target.Childs {
			if strings.Contains(url, uts.Url) {
				target = uts
				break
			}
			if i == len(target.Childs)-1 {
				return nil
			}
		}
	}
}

func (r *UrlTreeStruct) FindByTreeIter(treeIter *gtk.TreeIter) *UrlTreeStruct {
	if r.TreeIter.GtkTreeIter == treeIter.GtkTreeIter {
		return r
	}
	for _, uts := range r.Childs {
		found := uts.FindByTreeIter(treeIter)
		if found != nil {
			return found
		}
	}
	return nil
}

func (r *UrlTreeStruct) AppendAccordingUrl(newChild *UrlTreeStruct) bool {
	target := r
	for {
		if newChild == nil || newChild == r || r.Parent == newChild {
			return false
		}
		if len(target.Childs) == 0 {
			target.AppendChild(newChild)
			return true
		}
		for i, c := range target.Childs {
			if strings.Contains(newChild.Url, c.Url) {
				target = c
				break
			} else if i == len(target.Childs)-1 {
				target.AppendChild(newChild)
				return true
			}
		}
	}
}
