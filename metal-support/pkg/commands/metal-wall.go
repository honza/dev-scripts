package commands

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/openshift-metal3/dev-scripts/metal-releases/pkg/jobs"
)

type MetalWallCommand struct {
	port       string
	versions   []string
	builds     map[string]*jobs.Build // build id -> build
	BuildsInfo map[string][]BuildInfo // version -> info
}

type BuildInfo struct {
	Version            string
	JobName            string
	BuildId            string
	IsBlocking         bool
	Passed             bool
	NewBuildInProgress bool
	Url                string
}

func NewMetalWallCommand(port string) Command {
	return MetalWallCommand{
		port:       port,
		versions:   []string{"4.11", "4.10", "4.9", "4.8"},
		builds:     make(map[string]*jobs.Build),
		BuildsInfo: make(map[string][]BuildInfo),
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
	  {{range $version, $builds := .BuildsInfo}}
	    <div style="padding: 10px; border: 1px solid black;"> 
			<b>{{$version}}</b>
			{{range $builds}}
				{{if .Passed }}
				<div style="background-color: #cfc ; padding: 10px; border: 1px solid green;"> 
				{{else}}
				<div style="background-color: #fcc ; padding: 10px; border: 1px solid red;"> 
				{{if .IsBlocking}}<b>&#9888;</b>{{end}}
				{{end}}
				{{if .NewBuildInProgress}}<b>*</b>{{end}}
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
	mw.refreshData()
	mw.renderPage(w, r)
}

func (mw *MetalWallCommand) refreshData() {

	start := time.Now()

	defer func() {
		end := time.Now()
		log.Printf("Refresh data completed in %0.2f seconds\n", end.Sub(start).Seconds())
	}()

	for v, infos := range mw.BuildsInfo {
		for i, info := range infos {
			b := mw.builds[info.BuildId]

			if b == nil {
				log.Printf("[WARN] info.BuildId %s (%s) is nil! \n", info.JobName, info.BuildId)
				continue
			}

			latest, err := b.Job().GetLatestBuild()
			if err != nil {
				log.Printf("Unable to get latest build for %s (%s). Error: %s\n", info.JobName, info.BuildId, err)
				continue
			}
			// Check if there's a new build for the job
			if b.Id() != latest.Id() {
				err = latest.LoadCurrentStatus()
				if err != nil {
					log.Printf("Unable to get latest build info for %s (%s). Error: %s\n", info.JobName, info.BuildId, err)
					continue
				}

				// Update view
				if !latest.IsFinished() {
					log.Printf("Found new build for job %s (%s) in progress\n", b.Job().Name(), latest.Id())
					info.NewBuildInProgress = true
				} else {
					log.Printf("Found new completed build for job %s (%s) ", b.Job().Name(), latest.Id())
					// Update builds map
					delete(mw.builds, info.BuildId)
					mw.builds[latest.Id()] = latest

					info.NewBuildInProgress = false
					info.BuildId = latest.Id()
					info.Passed = latest.Passed()
					info.Url = latest.Url()
				}
				mw.BuildsInfo[v][i] = info
			}
		}
	}
}

// Gets the current latest completed build
func (mw *MetalWallCommand) fetchInitialData() error {

	for _, v := range mw.versions {
		var infos []BuildInfo
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

				mw.builds[b.Id()] = b

				infos = append(infos, BuildInfo{
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
		mw.BuildsInfo[v] = infos
	}

	return nil
}
