package entity

type Group struct {
	ID          int
	FullPath    string
	WebURL      string
	Description string
	Visibility  string
	Projects    []string
	Subgroups   []string
	MemberCount int
}

type Project struct {
	ID           int
	Name         string
	FullPath     string
	Description  string
	Visibility   string
	WebURL       string
	LastCommit   *Commit
	LastPipeline *Pipeline
}

type Commit struct {
	SHA       string
	Message   string
	Author    string
	CreatedAt string
}

type Pipeline struct {
	ID        int
	Status    string
	WebURL    string
	CreatedAt string
}

type Notification struct {
	Title   string
	Message string
}

type ProjectAction string

const (
	ProjectCreated   ProjectAction = "created"
	ProjectUpdated   ProjectAction = "updated"
	ProjectUnchanged ProjectAction = "unchanged"
)

type ProjectResult struct {
	Project *Project
	Action  ProjectAction
}

type NotifierEvent string

const (
	EventPipelineRunning  NotifierEvent = "pipeline_running"
	EventPipelineSuccess  NotifierEvent = "pipeline_success"
	EventPipelineFailed   NotifierEvent = "pipeline_failed"
	EventPipelineCanceled NotifierEvent = "pipeline_canceled"
	EventNewCommit        NotifierEvent = "new_commit"
)

// ProjectNotificationState tracks what was last notified for a given project.
type ProjectNotificationState struct {
	LastNotifiedPipelineID     int
	LastNotifiedPipelineStatus string
	LastNotifiedCommitSHA      string
}
