package classifier

type IntentVerdict string

const (
	IntentVerdictYes     IntentVerdict = "yes"
	IntentVerdictPartial IntentVerdict = "partial"
	IntentVerdictNo      IntentVerdict = "no"
)

type OptimalityVerdict string

const (
	OptimalityVerdictOptimal    OptimalityVerdict = "optimal"
	OptimalityVerdictAcceptable OptimalityVerdict = "acceptable"
	OptimalityVerdictSuboptimal OptimalityVerdict = "suboptimal"
	OptimalityVerdictUnknown    OptimalityVerdict = "unknown"
)

type MergeReadiness string

const (
	MergeReadinessReadyToMerge     MergeReadiness = "ready_to_merge"
	MergeReadinessWIP              MergeReadiness = "wip"
	MergeReadinessNeedsHumanReview MergeReadiness = "needs_human_review"
)

type FocusPriority string

const (
	FocusPriorityHigh   FocusPriority = "high"
	FocusPriorityMedium FocusPriority = "medium"
	FocusPriorityLow    FocusPriority = "low"
)

type IntentUnderstanding struct {
	Verdict          IntentVerdict `json:"verdict"`
	Confidence       float64       `json:"confidence"`
	Reason           string        `json:"reason"`
	UnderstoodIntent *string       `json:"understood_intent"`
}

type OptimalityAssessment struct {
	Verdict      OptimalityVerdict `json:"verdict"`
	Reason       string            `json:"reason"`
	Alternatives []string          `json:"alternatives"`
}

type FocusArea struct {
	Path     string        `json:"path"`
	Why      string        `json:"why"`
	Priority FocusPriority `json:"priority"`
}

type FirstPassReview struct {
	IntentUnderstanding IntentUnderstanding  `json:"intent_understanding"`
	Optimality          OptimalityAssessment `json:"optimality"`
	MergeReadiness      MergeReadiness       `json:"merge_readiness"`
	FocusAreas          []FocusArea          `json:"focus_areas"`
	BlockingQuestions   []string             `json:"blocking_questions"`
	RunID               string               `json:"run_id"`
}
