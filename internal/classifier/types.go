package classifier

type Classification string

const (
	ClassificationHumanRequired Classification = "human_required"
	ClassificationNoHuman      Classification = "no_human"
)

type Decision struct {
	Classification Classification `json:"classification"`
	Confidence     float64        `json:"confidence"`
	Reason         string         `json:"reason"`
	RunID          string         `json:"run_id"`
}

type PullRequestRef struct {
	Owner  string
	Repo   string
	Number string
	URL    string
}
