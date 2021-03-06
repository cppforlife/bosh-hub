package controllers

import (
	"net/http"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	mart "github.com/go-martini/martini"
	martrend "github.com/martini-contrib/render"

	bhjobsrepo "github.com/bosh-io/web/release/jobsrepo"
	bhrelsrepo "github.com/bosh-io/web/release/releasesrepo"
	bhjobui "github.com/bosh-io/web/ui/job"
	bhrelui "github.com/bosh-io/web/ui/release"
)

type JobsController struct {
	releasesRepo        bhrelsrepo.ReleasesRepository
	releaseVersionsRepo bhrelsrepo.ReleaseVersionsRepository
	jobsRepo            bhjobsrepo.JobsRepository

	showTmpl  string
	errorTmpl string

	logTag string
	logger boshlog.Logger
}

func NewJobsController(
	releasesRepo bhrelsrepo.ReleasesRepository,
	releaseVersionsRepo bhrelsrepo.ReleaseVersionsRepository,
	jobsRepo bhjobsrepo.JobsRepository,
	logger boshlog.Logger,
) JobsController {
	return JobsController{
		releasesRepo:        releasesRepo,
		releaseVersionsRepo: releaseVersionsRepo,
		jobsRepo:            jobsRepo,

		showTmpl:  "jobs/show",
		errorTmpl: "error",

		logTag: "JobsController",
		logger: logger,
	}
}

func (c JobsController) Show(req *http.Request, r martrend.Render, params mart.Params) {
	relSource, relVersion, jobName, err := c.extractShowParams(req, params)
	if err != nil {
		r.HTML(500, c.errorTmpl, err)
		return
	}

	c.logger.Debug(c.logTag, "Release source '%s'", relSource)

	var relVerRec bhrelsrepo.ReleaseVersionRec

	if len(relVersion) > 0 {
		relVerRec, err = c.releasesRepo.Find(relSource, relVersion)
		if err != nil {
			r.HTML(500, c.errorTmpl, err)
			return
		}
	} else {
		relVerRec, err = c.releasesRepo.FindLatest(relSource)
		if err != nil {
			r.HTML(500, c.errorTmpl, err)
			return
		}
	}

	rel, err := c.releaseVersionsRepo.Find(relVerRec)
	if err != nil {
		r.HTML(500, c.errorTmpl, err)
		return
	}

	relJobs, err := c.jobsRepo.FindAll(relVerRec)
	if err != nil {
		r.HTML(500, c.errorTmpl, err)
		return
	}

	viewRel := bhrelui.NewRelease(relVerRec, rel)

	for _, relJob := range relJobs {
		if relJob.Name == jobName {
			viewJob := bhjobui.NewJob(relJob, viewRel)
			viewJob.Release.NavPrimary = viewJob.Release.BuildNavigation(viewJob.URL())

			r.HTML(200, c.showTmpl, viewJob)
			return
		}
	}

	err = bosherr.Errorf("Release job '%s' is not found", jobName)
	r.HTML(404, c.errorTmpl, err)
}

func (c JobsController) extractShowParams(req *http.Request, params mart.Params) (string, string, string, error) {
	relSource := req.URL.Query().Get("source")

	if len(relSource) == 0 {
		return "", "", "", bosherr.Error("Param 'source' must be non-empty")
	}

	relVersion := req.URL.Query().Get("version")

	jobName := params["name"]

	if len(jobName) == 0 {
		return "", "", "", bosherr.Error("Param 'name' must be non-empty")
	}

	return relSource, relVersion, jobName, nil
}
