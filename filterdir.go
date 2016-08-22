/*
Package filterdir is a http.FileSystem middleware which can be used to provide a
filtered view of the filesystem to packages such as shurcooL/vfsgen. FilterDir
was developed with the primary use case of assisting with the packaging of assets
for Single-Page Applications (SPAs) such as Angular or Ember, directly into Go
source code.

As anyone that has developed a JavaScript application knows, when using a larger
library such as RxJS there are hundreds of files in the library, but your application
may only need 10 of them to support the feature set your are using.  To make our
applications more performant in the browser, RxJS allows you to target your imports
to only include what you need.

When bundling up your application into your Go source, it can be quite painful to
figure out exactly which files you need for your application to run, but given
the size of the numerous javascript libraries that will surely exist in your application,
your Go binary will get very big carrying along all the unneeded files.

FilterDir makes it a snap to obtain an exact list of files being used and run
vfsgen to bundle them up for you!

There are two modes of operation:

	* With FilterMode = false, all files are visible to the Open() method and a list
	  of files are recorded as they are accessed.  Using this mode an IncludeList can
	  be generated, detailing the exact set of files being used by the http.FileSystem.

	* With FilterMode = true, only files contained in IncludeList will be visible
	  to the Open() method. Using this mode with vfsgen, the exact list of files
	  needed can be packaged into go source code.

Given a simple http.FileServer

main.go

	package main

	import "net/http"

	func main() {

		http.Handle("/", http.FileServer(http.Dir("gui")))

		http.ListenAndServe("localhost:8080", nil)
	}

all files in the "gui" directory in the same folder as our Go program will be served
on localhost:8080.

To make use of FilterDir, create a file as follows:

assets.go

	// +build dev

	package main

	import "github.com/jtwatson/filterdir"

	var assets = filterdir.New("gui", filterdir.Options{})

Notice the "dev" build tag. Now with one small change to our original http.FileServer
we can use "assets" as follows:

main.go

	package main

	import "net/http"

	func main() {

		http.Handle("/", http.FileServer(assets))

		http.ListenAndServe("localhost:8080", nil)
	}

Now we can run our program:

	go run -tags=dev main.go assets.go

As soon as FilterDir receives its first request, it will start a console application
that displays a summary of the files requested. It also has options as follows:

	c : Clear all files from IncludeList
	s : Save current file list to go source
	g : Generate go code that statically implements all files in list (using shurcooL/vfsgen)
	q : Quit

*/
package filterdir

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"golang.org/x/net/context"

	"github.com/jroimartin/gocui"
	"github.com/shurcooL/vfsgen"
)

// FilterDir is a http.FileSystem middleware that implements filtering of
// files, which restrict visible files to those in the IncludeList.
type FilterDir struct {
	loadOnce  sync.Once
	startOnce sync.Once
	dir       http.Dir
	options   Options
	requests  chan string
	include   map[string]struct{}

	// FilterMode enables the filter so only files found in IncludeList
	// will be returned.
	FilterMode bool

	// IncludeList is a slice of files that are allowed to be returned when
	// FilterMode is set to true.
	IncludeList []string
}

// New returns a newly instanciated FilterDir with dir as the root directory used to server files.
func New(dir string, opt Options) *FilterDir {
	opt.fillMissing()
	return &FilterDir{dir: http.Dir(dir), options: opt, requests: make(chan string, 100)}
}

// Options used by vfsgen when generating the statically implemented virtual filesystem.
func (f *FilterDir) Options() vfsgen.Options {
	return vfsgen.Options{
		Filename:        f.options.Filename,
		PackageName:     f.options.PackageName,
		BuildTags:       f.options.VfsgenBuildTags,
		VariableName:    f.options.VariableName,
		VariableComment: f.options.VariableComment,
	}
}

// Open attempts to open name, which is a resource under the root dir provided to FilterDir
func (f *FilterDir) Open(name string) (http.File, error) {
	file, err := f.dir.Open(name)
	if err != nil {
		return nil, err
	}
	if f.FilterMode == false {
		f.startOnce.Do(func() {
			go f.startGUI()
		})
		f.requests <- name
		return file, nil
	}

	// We are in FilterMode, so results will be filtered
	f.loadOnce.Do(f.loadIncludeList)

	if _, ok := f.include[name]; ok {
		return &File{File: file, name: name, include: f.include}, nil
	}

	return nil, os.ErrNotExist
}

func (f *FilterDir) loadIncludeList() {
	f.include = make(map[string]struct{})
	f.include["/"] = struct{}{}
	for _, file := range f.IncludeList {
		f.include[file] = struct{}{}
		dirs := strings.Split(file, "/")
		for i := 2; i < len(dirs); i++ {
			f.include[strings.Join(dirs[:i], "/")] = struct{}{}
		}
	}
}

func (f *FilterDir) startGUI() {

	// Process incoming file requests
	reqs := processRequests(f.IncludeList, f.requests)

	// Create GUI
	gui := gocui.NewGui()
	if err := gui.Init(); err != nil {
		log.Fatal(err)
	}
	defer gui.Close()

	// Draw UI
	gui.SetLayout(layout)
	gui.Cursor = true

	// Push file list changes to UI
	ctx, done := context.WithCancel(context.Background())
	defer done()
	go pushUpdates(ctx, gui, reqs)

	// Wire up keys to actions
	if err := bindKeys(gui, reqs, f); err != nil {
		log.Fatal(err)
	}

	// Run GUI
	if err := gui.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatal(err)
	}
}

func (f *FilterDir) saveList(list []string) error {
	f.IncludeList = list

	// Create output file.
	lf, err := os.Create(f.options.ListFileName)
	if err != nil {
		return err
	}
	defer lf.Close()

	err = t.ExecuteTemplate(lf, "Header", f.options)
	if err != nil {
		return err
	}

	for _, l := range list {
		err = t.ExecuteTemplate(lf, "Files", l)
		if err != nil {
			return err
		}
	}

	err = t.ExecuteTemplate(lf, "Footer", f.options)
	if err != nil {
		return err
	}

	return nil
}

func (f *FilterDir) generateAssets(list []string) error {
	f.IncludeList = list
	f.FilterMode = true

	err := vfsgen.Generate(f, f.Options())
	if err != nil {
		return err
	}
	return nil
}

// A File is returned by a FileSystem's Open method and can be
// served by the FileServer implementation.
//
// The methods should behave the same as those on an *os.File.
type File struct {
	http.File
	name    string
	include map[string]struct{}
}

// Readdir behaves the same way as os.File.Readdir, but additionally
// filters on IncludeList
func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	// Remove trailing '/' if it is present
	if f.name[len(f.name)-1:] == "/" {
		f.name = f.name[:len(f.name)-1]
	}
	info, err := f.File.Readdir(count)
	var newInfo []os.FileInfo
	for _, i := range info {
		if _, ok := f.include[f.name+"/"+i.Name()]; ok {
			newInfo = append(newInfo, i)
		}
	}
	return newInfo, err
}

// Options for code generation.
type Options struct {
	// Filename of the generated Go code output (including extension).
	// If left empty, it defaults to "{{toLower .VariableName}}_vfsdata.go".
	Filename string

	// PackageName is the name of the package in the generated code.
	// If left empty, it defaults to "main".
	PackageName string

	// VfsgenBuildTags are the optional build tags in the generated code.
	// If left empty, it defaults to "!dev".
	// The build tags syntax is specified by the go tool.
	VfsgenBuildTags string

	// VariableName is the name of the http.FileSystem variable in the generated code.
	// If left empty, it defaults to "assets".
	VariableName string

	// VariableComment is the comment of the http.FileSystem variable in the generated code.
	// If left empty, it defaults to "{{.VariableName}} statically implements the virtual filesystem provided to vfsgen.".
	VariableComment string

	// ListFileName is the name of the go source file which holds the generated code for IncludeList.
	// If left empty, it defaults to "assets_list.go".
	ListFileName string

	// ListFileBuildTags are the optional build tags in the generated code for IncludeList.
	// If left empty, it defaults to "dev".
	// The build tags syntax is specified by the go tool.
	ListFileBuildTags string
}

// fillMissing sets default values for mandatory options that are left empty.
func (opt *Options) fillMissing() {
	if opt.PackageName == "" {
		opt.PackageName = "main"
	}
	if opt.VariableName == "" {
		opt.VariableName = "assets"
	}
	if opt.ListFileName == "" {
		opt.ListFileName = "assets_list.go"
	}
	if opt.VfsgenBuildTags == "" {
		opt.VfsgenBuildTags = "!dev"
	}
	if opt.ListFileBuildTags == "" {
		opt.ListFileBuildTags = "dev"
	}
}

func processRequests(savedIncludeList []string, requests chan string) *sortedList {
	var changed bool
	qchan := make(chan struct{})
	achan := make(chan []string)
	cchan := make(chan bool)
	clear := make(chan struct{})

	includeList := make([]string, 0, 100)
	includeMap := make(map[string]bool)

	go func() {
		for {
			select {
			case r := <-requests:
				if includeMap[r] == false {
					includeMap[r] = true
					includeList = append(includeList, r)
					changed = true
				}
			case <-qchan:
				if changed {
					// sort includeList
					sort.StringSlice(includeList).Sort()
					changed = false
				}
				sortedList := make([]string, len(includeList))
				copy(sortedList, includeList)
				achan <- sortedList
			case cchan <- changed:
			case <-clear:
				includeList = make([]string, 0, 100)
				includeMap = make(map[string]bool)
				changed = true
			}
		}
	}()

	return &sortedList{q: qchan, a: achan, chg: cchan, c: clear}
}

type sortedList struct {
	q   chan struct{}
	a   chan []string
	chg chan bool
	c   chan struct{}
}

func (l *sortedList) List() []string {
	l.q <- struct{}{}
	return <-l.a
}

func (l *sortedList) Changed() bool {
	return <-l.chg
}

func (l *sortedList) Clear() {
	l.c <- struct{}{}
}

var t = template.Must(template.New("").Parse(`{{define "Header"}}// Code generated by FilterDir

{{with .ListFileBuildTags}}// +build {{.}}

{{end}}package {{.PackageName}}

func init() {
	{{.VariableName}}.IncludeList = []string{
{{end}}

{{define "Files"}}		"{{.}}",
{{end}}

{{define "Footer"}}	}
}
{{end}}
`))
