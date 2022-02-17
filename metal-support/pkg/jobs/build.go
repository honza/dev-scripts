package jobs

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"time"
)

type Started struct {
	Timestamp int64 `json:"timestamp"`
}

type Finished struct {
	Timestamp int64  `json:"timestamp"`
	Passed    bool   `json:"passed"`
	Result    string `json:"result"`
	Revision  string `json:"revision"`
}

type Build struct {
	// The job owner of this build
	job *Job
	// The unique build id
	id string
	// The starting info of the build
	started Started
	// The end status of the build
	finished *Finished
	// A link to the build
	buildUrl string
	// A link to the build steps artifacts
	artifactsUrl string
}

func (b *Build) Id() string {
	return b.id
}

func (b *Build) IsFinished() bool {
	return b.finished != nil
}

func (b *Build) Passed() bool {
	return b.finished.Passed
}

func (b *Build) Url() string {
	return b.buildUrl
}

func (b *Build) Finished() time.Time {
	return time.Unix(b.finished.Timestamp, 0)
}

func (b *Build) Job() *Job {
	return b.job
}

func (b *Build) LoadCurrentStatus() error {

	started, err := FetchRemoteFile(fmt.Sprintf("%s/started.json", b.buildUrl))
	if err != nil {
		return err
	}
	err = json.Unmarshal(started, &b.started)
	if err != nil {
		return err
	}

	finished, err := FetchRemoteFile(fmt.Sprintf("%s/finished.json", b.buildUrl))
	if err != nil {
		return err
	}

	var f Finished
	err = json.Unmarshal(finished, &f)
	// If the build is still pending, the finished.json file is not published
	if err == nil {
		b.finished = &f
	}
	return nil
}

// LoadTestResults fetches the test results related to the current build
func (b *Build) LoadTestResults() (*TestSuite, error) {
	testsUrl := fmt.Sprintf("%s/%s/%s/artifacts/%s/baremetalds-e2e-test/artifacts/junit/", baseArtifactsUrl, b.job.name, b.id, b.job.safeName)

	suite := TestSuite{}
	testsFilename, err := b.getTestResultsFilename(testsUrl)
	if err != nil {
		// In some cases the test step could fail before running the tests
		// so the results artifacts are not published. This build will be
		// skipped
		return &suite, nil
	}

	tests, err := FetchRemoteFile(fmt.Sprintf("%s/%s", testsUrl, testsFilename))
	if err != nil {
		return nil, err
	}
	err = xml.Unmarshal(tests, &suite)
	if err != nil {
		return nil, err
	}

	return &suite, nil
}

func (b *Build) getTestResultsFilename(url string) (string, error) {
	s := NewHtmlScraper(url, `.*/(junit_e2e.*\.xml)`)
	res, err := s.Get()
	if err != nil {
		return "", err
	}

	if len(res) == 0 {
		return "", errors.New("not found")
	}

	return res[0], nil
}

func NewBuild(id string, job *Job) *Build {
	return &Build{
		id:           id,
		job:          job,
		buildUrl:     fmt.Sprintf("%s/%s/%s", baseArtifactsUrl, job.name, id),
		artifactsUrl: fmt.Sprintf("%s/%s/%s/artifacts/%s", baseArtifactsUrl, job.name, id, job.safeName),
	}
}
