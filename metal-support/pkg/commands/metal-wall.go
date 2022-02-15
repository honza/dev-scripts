package commands

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/openshift-metal3/dev-scripts/metal-releases/pkg/jobs"
)

type MetalWallCommand struct {
	port string

	Builds map[string][]BuildInfo
}

type BuildInfo struct {
	Version    string
	JobName    string
	BuildId    string
	IsBlocking bool
	Passed     bool
	Url        string
}

func NewMetalWallCommand(port string) Command {
	return MetalWallCommand{
		port:   port,
		Builds: make(map[string][]BuildInfo),
	}
}

func (mw MetalWallCommand) Run() error {

	log.Println("Fetching current status...")
	mw.fetchInitialData()

	log.Println("Launching metal wall server at port", mw.port)
	http.HandleFunc("/", mw.MetalWallHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", mw.port), nil))

	return nil
}

func (mw *MetalWallCommand) template() string {
	return `
<html>
	<head>
	  <title>Metal Wall2</title>
	  <script>
	  function autoRefresh() {
		  window.location = window.location.href;
	  }
	  setInterval('autoRefresh()', 30000);
  </script>
	</head>
	<body>
	  <h2>Metal Wall</h2>
	  {{range $version, $builds := .Builds}}
	    <div style="padding: 10px; border: 1px solid black;"> 
			<b>{{$version}}</b>
			{{range $builds}}
				{{if .Passed }}
				<div style="background-color: #cfc ; padding: 10px; border: 1px solid green;"> 
				{{else}}
				<div style="background-color: #fcc ; padding: 10px; border: 1px solid red;"> 
				{{if .IsBlocking}}<b>&#9888;</b>{{end}}
				{{end}}
				{{.JobName}} (<a href="{{.Url}}">{{.BuildId}}</a>)
				</div>
			{{end}}
		</div>
	  {{end}}
	</body>
  </html>
`
}

func (mw *MetalWallCommand) BuildId(args ...interface{}) string {
	return "fixed"
}

func (mw *MetalWallCommand) renderPage(w http.ResponseWriter, r *http.Request) {
	t := template.New("metal wall template")
	t.Funcs(template.FuncMap{
		"buildId": mw.BuildId,
	})
	t, err := t.Parse(mw.template())
	if err != nil {
		log.Fatal(err)
	}

	t.Execute(w, mw)
}

func (mw *MetalWallCommand) MetalWallHandler(w http.ResponseWriter, r *http.Request) {
	//blocking, _ := mw.refreshData()
	mw.renderPage(w, r)
}

// Gets the current latest completed build
func (mw *MetalWallCommand) fetchInitialData() error {

	versions := []string{"4.10", "4.11"}

	for _, v := range versions {
		var builds []BuildInfo
		for _, j := range jobs.BlockingJobs(v) {

			buildIds, err := j.FetchAllBuildIds()
			if err != nil {
				return err
			}

			for _, id := range buildIds {
				b := jobs.NewBuild(id, j)
				b.LoadCurrentStatus()

				if !b.IsFinished() {
					continue
				}

				builds = append(builds, BuildInfo{
					Version:    v,
					JobName:    j.SafeName(),
					BuildId:    b.Id(),
					Url:        b.Url(),
					Passed:     b.Passed(),
					IsBlocking: true,
				})
				break
			}
		}
		mw.Builds[v] = builds
	}

	return nil
}
