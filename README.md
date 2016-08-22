

# filterdir
`import "github.com/jtwatson/filterdir"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)

## <a name="pkg-overview">Overview</a>
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




## <a name="pkg-index">Index</a>
* [type File](#File)
  * [func (f *File) Readdir(count int) ([]os.FileInfo, error)](#File.Readdir)
* [type FilterDir](#FilterDir)
  * [func New(dir string, opt Options) *FilterDir](#New)
  * [func (f *FilterDir) Open(name string) (http.File, error)](#FilterDir.Open)
  * [func (f *FilterDir) Options() vfsgen.Options](#FilterDir.Options)
* [type Options](#Options)


#### <a name="pkg-files">Package files</a>
[filterdir.go](/src/github.com/jtwatson/filterdir/filterdir.go) [termtool.go](/src/github.com/jtwatson/filterdir/termtool.go) 






## <a name="File">type</a> [File](/src/target/filterdir.go?s=6530:6606#L247)
``` go
type File struct {
    http.File
    // contains filtered or unexported fields
}
```
A File is returned by a FileSystem's Open method and can be
served by the FileServer implementation.

The methods should behave the same as those on an *os.File.










### <a name="File.Readdir">func</a> (\*File) [Readdir](/src/target/filterdir.go?s=6703:6759#L255)
``` go
func (f *File) Readdir(count int) ([]os.FileInfo, error)
```
Readdir behaves the same way as os.File.Readdir, but additionally
filters on IncludeList




## <a name="FilterDir">type</a> [FilterDir](/src/target/filterdir.go?s=3145:3541#L100)
``` go
type FilterDir struct {

    // FilterMode enables the filter so only files found in IncludeList
    // will be returned.
    FilterMode bool

    // IncludeList is a slice of files that are allowed to be returned when
    // FilterMode is set to true.
    IncludeList []string
    // contains filtered or unexported fields
}
```
FilterDir is a http.FileSystem middleware that implements filtering of
files, which restrict visible files to those in the IncludeList.







### <a name="New">func</a> [New](/src/target/filterdir.go?s=3642:3686#L118)
``` go
func New(dir string, opt Options) *FilterDir
```
New returns a newly instanciated FilterDir with dir as the root directory used to server files.





### <a name="FilterDir.Open">func</a> (\*FilterDir) [Open](/src/target/filterdir.go?s=4272:4328#L135)
``` go
func (f *FilterDir) Open(name string) (http.File, error)
```
Open attempts to open name, which is a resource under the root dir provided to FilterDir




### <a name="FilterDir.Options">func</a> (\*FilterDir) [Options](/src/target/filterdir.go?s=3887:3931#L124)
``` go
func (f *FilterDir) Options() vfsgen.Options
```
Options used by vfsgen when generating the statically implemented virtual filesystem.




## <a name="Options">type</a> [Options](/src/target/filterdir.go?s=7111:8385#L271)
``` go
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
```
Options for code generation.














- - -
Generated by [godoc2md](http://godoc.org/github.com/davecheney/godoc2md)
