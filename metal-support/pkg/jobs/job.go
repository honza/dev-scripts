package jobs

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

const (
	// This is the url where the Prow jobs artifacts are stored
	baseArtifactsUrl = "https://gcsweb-ci.apps.ci.l2s4.p1.openshiftapps.com/gcs/origin-ci-test/logs"

	// Max number of stored builds per job
	maxBuilds = 20
)

var (
	blockingJobs []string = []string{
		"periodic-ci-openshift-release-master-nightly-%s-e2e-metal-ipi",
		"periodic-ci-openshift-release-master-nightly-%s-e2e-metal-ipi-ovn-ipv6",
		"periodic-ci-openshift-release-master-nightly-%s-e2e-metal-ipi-serial-ipv4",
	}
	informingJobs []string = []string{
		"periodic-ci-openshift-release-master-nightly-%s-e2e-metal-ipi-virtualmedia",
		"periodic-ci-openshift-release-master-nightly-%s-e2e-metal-ipi-compact",
		"periodic-ci-openshift-release-master-nightly-%s-e2e-metal-ipi-ovn-dualstack",
		"periodic-ci-openshift-release-master-nightly-%s-e2e-metal-ipi-serial-ipv6",
		"periodic-ci-openshift-release-master-nightly-%s-e2e-metal-ipi-serial-virtualmedia",
		"periodic-ci-openshift-release-master-nightly-%s-e2e-metal-ipi-serial-compact",
		"periodic-ci-openshift-release-master-nightly-%s-e2e-metal-ipi-serial-ovn-dualstack",
	}
)

// BlockingJobs returns a list of blocking jobs for the specified version
func BlockingJobs(version string) (jobs []*Job) {
	// For the sake of simplicity, let's use an hard-coded list
	return makeJobs(version, blockingJobs)
}

// InformingJobs returns a list of informing jobs for the specified version (upgrades excluded)
func InformingJobs(version string) (jobs []*Job) {
	// For the sake of simplicity, let's use an hard-coded list
	return makeJobs(version, informingJobs)
}

func makeJobs(version string, jobsTemplate []string) (jobs []*Job) {
	for _, j := range jobsTemplate {
		jobs = append(jobs, NewJob(fmt.Sprintf(j, version), version))
	}
	return
}

// NewJob creates a new job instance
func NewJob(name, version string) *Job {
	return &Job{
		name:     name,
		safeName: name[strings.Index(name, "e2e"):],
		version:  version,
		url:      fmt.Sprintf("%s/%s/", baseArtifactsUrl, name),
		builds:   []*Build{},
		history: JobHistory{
			Data: make(map[string]TestHistory),
		},
	}
}

// TestHistory is used to accumulate the detected flakes for given test
type TestHistory struct {
	PreviousState bool
	Flakes        float32
}

// JobHistory keeps all the relevant info for the analyzed builds
// for a given job
type JobHistory struct {
	From        int64
	To          int64
	TotalBuilds float32
	Data        map[string]TestHistory
}

// Job represent a Prow job
type Job struct {
	name     string
	safeName string
	version  string
	url      string
	builds   []*Build
	history  JobHistory
}

func (j *Job) Name() string {
	return j.name
}

func (j *Job) SafeName() string {
	return j.safeName
}

func (j *Job) FetchAllBuildIds() (buildIds []string, err error) {
	s := NewHtmlScraper(j.url, `.*/(\d+)/`)
	buildIds, err = s.Get()
	if err != nil {
		return nil, err
	}

	sort.Slice(buildIds, func(i, j int) bool {
		return buildIds[i] > buildIds[j]
	})
	return buildIds, nil
}

func (j *Job) GetLatestBuild() (*Build, error) {
	latest, err := FetchRemoteFile(fmt.Sprintf("%s/latest-build.txt", j.url))
	if err != nil {
		return nil, err
	}

	return NewBuild(string(latest), j), nil
}

func (j *Job) GetBuildsSince(from string) error {
	log.Println("--------------------------------------------------")
	log.Println(j.name, "Listing builds")
	since, err := time.Parse(time.RFC3339, fmt.Sprintf("%sT00:00:00Z", from))
	if err != nil {
		return err
	}

	buildIds, err := j.FetchAllBuildIds()
	if err != nil {
		return err
	}

	for _, id := range buildIds {
		b := NewBuild(id, j)
		b.LoadCurrentStatus()

		if !b.IsFinished() {
			continue
		}

		if b.Finished().Before(since) {
			break
		}

		j.builds = append(j.builds, b)

		if len(j.builds) >= maxBuilds {
			break
		}

	}

	log.Println(j.name, fmt.Sprintf("Found %d builds since %s", len(j.builds), from))
	return nil
}

func (j *Job) LookForIntermittentFailures() error {
	for _, b := range j.builds {
		suite, err := b.LoadTestResults()
		if err != nil {
			return err
		}

		for _, tc := range suite.TestCases {

			if tc.Ignore() {
				continue
			}

			thc, ok := j.history.Data[tc.Name]
			if !ok {
				thc = TestHistory{
					PreviousState: true,
				}
			}

			if tc.IsPassed() != thc.PreviousState {
				thc.Flakes += 0.5
			}
			thc.PreviousState = tc.IsPassed()

			j.history.Data[tc.Name] = thc
		}

		if len(suite.TestCases) > 0 {
			j.history.TotalBuilds += 1.0
		}
	}

	j.history.To = j.builds[0].finished.Timestamp
	j.history.From = j.builds[len(j.builds)-1].finished.Timestamp

	return nil
}

func (j *Job) ShowIntermittentFailures() {

	type FlakyTest struct {
		name      string
		flakiness float32
	}

	flakes := []FlakyTest{}
	for k, v := range j.history.Data {
		if v.Flakes == 0.0 {
			continue
		}

		flakiness := v.Flakes / j.history.TotalBuilds
		flakes = append(flakes, FlakyTest{
			name:      k,
			flakiness: flakiness,
		})
	}

	sort.Slice(flakes, func(i, j int) bool {
		return flakes[i].flakiness > flakes[j].flakiness
	})

	to := time.Unix(j.history.To, 0).UTC()
	from := time.Unix(j.history.From, 0).UTC()
	log.Println(j.name, fmt.Sprintf("Top flaky tests (last %0.f days, %0.f builds)", to.Sub(from).Hours()/24, j.history.TotalBuilds))
	for _, f := range flakes {
		log.Printf("%0.2f\t%s\n", f.flakiness, f.name)
	}
}
