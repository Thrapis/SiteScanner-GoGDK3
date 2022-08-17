package main

import (
	"strings"
	"sync"
)

const (
	STATUS_NO_INFO = iota
	STATUS_SUCCESS
	STATUS_FAILURE
)

type UrlTreeStruct struct {
	Url        string
	Status     int
	Parent     *UrlTreeStruct
	Childs     []*UrlTreeStruct
	childMutex sync.Mutex
}

func NewUrlTreeStruct(url string) *UrlTreeStruct {
	return &UrlTreeStruct{Url: url, Status: STATUS_NO_INFO, Childs: make([]*UrlTreeStruct, 0)}
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
	if r.Url == url {
		return r
	}
	for _, uts := range r.Childs {
		finded := uts.FindByUrl(url)
		if finded != nil {
			return finded
		}
	}
	return nil
}
