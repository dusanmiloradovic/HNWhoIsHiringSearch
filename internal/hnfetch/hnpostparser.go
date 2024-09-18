package hnfetch

import (
	"strings"
)

type PostMeta interface {
	GetDescription() string
	GetCompany() *string
	GetJobTitle() *string // there is no consistency in HM posts, title includes everything before the last pipe symbol
	IsRemote() bool
}

type PostMetaInfo struct {
	Text string
}

func (p PostMetaInfo) IsRemote() bool {
	return strings.Index(strings.ToLower(p.Text), "remote") != -1
}

func (p PostMetaInfo) GetDescription() string {
	ind := strings.LastIndex(p.Text, "|")
	if ind == -1 {
		return p.Text
	}
	return p.Text[ind+1:]
}

func (p PostMetaInfo) GetCompany() *string {
	ind := strings.Index(p.Text, "|")
	if ind == -1 {
		return nil
	}
	company := p.Text[0:ind]
	return &company
}

func (p PostMetaInfo) GetJobTitle() *string {
	ind := strings.LastIndex(p.Text, "|")
	if ind == -1 {
		return nil
	}
	ret := p.Text[0:ind]
	return &ret
}

func NewPostMeta(description string) PostMeta {
	ret := PostMetaInfo{Text: description}
	return ret
}
