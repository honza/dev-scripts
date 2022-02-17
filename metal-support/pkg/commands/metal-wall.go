package commands

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/openshift-metal3/dev-scripts/metal-releases/pkg/jobs"
)

type MetalWallCommand struct {
	port        string
	versions    []string
	builds      map[string]*jobs.Build // build id -> build
	BuildsInfo  map[string][]BuildInfo // version -> info
	LastUpdated time.Time
}

type BuildInfo struct {
	Version            string `json:"version"`
	JobName            string `json:"job_name"`
	BuildId            string `json:"build_id"`
	IsBlocking         bool   `json:"is_blocking"`
	Passed             bool   `json:"passed"`
	NewBuildInProgress bool   `json:"new_build_in_progress"`
	Url                string `json:"url"`
}

type Version struct {
	Name   string      `json:"name"`
	Builds []BuildInfo `json:"builds"`
}

type JSONResponse struct {
	Versions    []Version `json:"versions"`
	LastUpdated time.Time `json:"last_updated"`
}

func NewMetalWallCommand(port string) Command {
	return MetalWallCommand{
		port:       port,
		versions:   []string{"4.11", "4.10", "4.9", "4.8"},
		builds:     make(map[string]*jobs.Build),
		BuildsInfo: make(map[string][]BuildInfo),
	}
}

func (mw MetalWallCommand) AsJSON() JSONResponse {
	versions := []Version{}
	for version, builds := range mw.BuildsInfo {
		versions = append(versions, Version{Name: version, Builds: builds})
	}
	return JSONResponse{Versions: versions, LastUpdated: mw.LastUpdated}
}

func (mw MetalWallCommand) Run() error {

	log.Println("Fetching current status...")
	mw.fetchInitialData()

	log.Println("Launching metal wall server at port", mw.port)
	http.HandleFunc("/", mw.MetalWallHandler)
	http.HandleFunc("/data.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mw.AsJSON())
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", mw.port), nil))

	return nil
}

func (mw *MetalWallCommand) template() string {

	contents, err := ioutil.ReadFile("../templates/index.html")
	if err != nil {
		log.Fatal(err)
	}
	return string(contents)
}

func (mw *MetalWallCommand) renderPage(w http.ResponseWriter, r *http.Request) {
	t := template.New("metal wall template")
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
	mw.LastUpdated = time.Now().UTC()
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

	mw.LastUpdated = time.Now().UTC()

	return nil
}
