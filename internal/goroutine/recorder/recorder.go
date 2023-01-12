package recorder

import (
	"encoding/json"
	"time"

	"github.com/sourcegraph/log"
	"github.com/sourcegraph/sourcegraph/internal/rcache"
	"github.com/sourcegraph/sourcegraph/lib/errors"
	"k8s.io/utils/strings/slices"
)

type Recordable interface {
	Start()
	Stop()
	Name() string
	Type() RoutineType
	JobName() string
	SetJobName(string)
	Description() string
	Interval() time.Duration
	RegisterRecorder(recorder *Recorder)
}

type Recorder struct {
	rcache      *rcache.Cache
	logger      log.Logger
	recordables []Recordable
	hostName    string
}

// seenTimeout is the maximum time we allow no activity for each host, job, and routine.
// After this time, we consider them non-existent.
const seenTimeout = 6 * 24 * time.Hour // 6 days

const keyPrefix = "background-job-logger"

// backgroundRoutine represents a single routine in a background job, and is used for serialization to/from Redis.
type backgroundRoutine struct {
	Name        string        `json:"name"`
	Type        RoutineType   `json:"type"`
	JobName     string        `json:"jobName"`
	Description string        `json:"description"`
	Interval    time.Duration `json:"interval"` // Assumes that the routine runs at a fixed interval across all hosts
	LastSeen    string        `json:"lastSeen"`
}

// New creates a new recorder.
func New(logger log.Logger, hostName string, cache *rcache.Cache) *Recorder {
	return &Recorder{rcache: cache, logger: logger, hostName: hostName}
}

// Register registers a new routine with the recorder.
func (m *Recorder) Register(r Recordable) {
	m.recordables = append(m.recordables, r)
}

// RegistrationDone should be called after all recordables have been registered.
// It saves the known job names, host names, and routine names in Redis, along with updating their “last seen” date/time.
func (m *Recorder) RegistrationDone() {
	// Save/update known job names
	for _, jobName := range m.collectAllJobNames() {
		m.saveKnownJobName(jobName)
	}

	// Save known host name
	m.saveKnownHostName()

	// Save/update all known recordables
	for _, r := range m.recordables {
		m.saveKnownRoutine(r)
	}
}

// collectAllJobNames collects all known job names in Redis.
func (m *Recorder) collectAllJobNames() []string {
	var allJobNames []string
	for _, r := range m.recordables {
		jobName := r.JobName()
		if slices.Contains(allJobNames, jobName) {
			continue
		}
		allJobNames = append(allJobNames, jobName)
	}

	return allJobNames
}

// saveKnownJobName updates the “lastSeen” date of a known job in Redis. Also adds it to the list of known jobs if it doesn’t exist.
func (m *Recorder) saveKnownJobName(jobName string) {
	err := m.rcache.SetHashItem("knownJobNames", jobName, time.Now().Format(time.RFC3339))
	if err != nil {
		m.logger.Error("failed to save/update known job name", log.Error(err), log.String("jobName", jobName))
	}
}

// saveKnownHostName updates the “lastSeen” date of a known host in Redis. Also adds it to the list of known hosts if it doesn’t exist.
func (m *Recorder) saveKnownHostName() {
	err := m.rcache.SetHashItem("knownHostNames", m.hostName, time.Now().Format(time.RFC3339))
	if err != nil {
		m.logger.Error("failed to save/update known host name", log.Error(err), log.String("hostName", m.hostName))
	}
}

// saveKnownRouting updates the routine in Redis. Also adds it to the list of known recordables if it doesn’t exist.
func (m *Recorder) saveKnownRoutine(recordable Recordable) {
	routine := backgroundRoutine{
		Name:        recordable.Name(),
		Type:        recordable.Type(),
		JobName:     recordable.JobName(),
		Description: recordable.Description(),
		Interval:    recordable.Interval(),
		LastSeen:    time.Now().Format(time.RFC3339),
	}

	// Serialize Routine
	routineJson, err := json.Marshal(routine)
	if err != nil {
		m.logger.Error("failed to serialize routine", log.Error(err), log.String("routineName", routine.Name))
		return
	}

	// Save/update Routine
	err = m.rcache.SetHashItem("knownRoutines", routine.Name, string(routineJson))
	if err != nil {
		m.logger.Error("failed to save/update known routine", log.Error(err), log.String("routineName", routine.Name))
	}
}

func (m *Recorder) LogStart(r Recordable) {
	m.rcache.Set(r.Name()+":"+m.hostName+":"+"lastStart", []byte(time.Now().Format(time.RFC3339)))
	m.logger.Debug("" + r.Name() + " just started! 🚀")
}

func (m *Recorder) LogStop(r Recordable) {
	m.rcache.Set(r.Name()+":"+m.hostName+":"+"lastStop", []byte(time.Now().Format(time.RFC3339)))
	m.logger.Debug("" + r.Name() + " just stopped! 🛑")
}

func (m *Recorder) LogRun(r Recordable, duration time.Duration, runErr error) {
	durationMs := int32(duration.Milliseconds())
	err := m.saveRun(r.Name(), m.hostName, durationMs, runErr)
	if err != nil {
		m.logger.Error("failed to save run", log.Error(err))
	}

	err = saveRunStats(m.rcache, r.Name(), durationMs, runErr != nil)
	if err != nil {
		m.logger.Error("failed to save run stats", log.Error(err))
	}

	m.logger.Debug("Hello from " + r.Name() + "! 😄")
}

func (m *Recorder) saveRun(routineName string, hostName string, durationMs int32, err error) error {
	errorMessage := ""
	stackTrace := ""
	if err != nil {
		errorMessage = err.Error()
		stackTrace = "stack trace goes here"
	}

	// Create Run
	run := RoutineRun{
		At:           time.Now(),
		HostName:     hostName,
		DurationMs:   durationMs,
		ErrorMessage: errorMessage,
		StackTrace:   stackTrace,
	}

	// Serialize run
	runJson, err := json.Marshal(run)
	if err != nil {
		return errors.Wrap(err, "serialize run")
	}

	// Save run
	err = m.rcache.AddToList(routineName+":"+hostName+":"+"recentRuns", string(runJson))
	if err != nil {
		return errors.Wrap(err, "save run")
	}

	return nil
}

// saveRunStats updates the run stats for a routine in Redis.
func saveRunStats(c *rcache.Cache, routineName string, durationMs int32, errored bool) error {
	// Prepare data
	isoDate := time.Now().Format("2006-01-02")

	// Get stats
	lastStatsRaw, found := c.Get(routineName + ":runStats:" + isoDate)
	var lastStats RoutineRunStats
	if found {
		err := json.Unmarshal(lastStatsRaw, &lastStats)
		if err != nil {
			return errors.Wrap(err, "deserialize last stats")
		}
	}

	// Update stats
	newStats := addRunToStats(lastStats, durationMs, errored)

	// Serialize and save updated stats
	updatedStatsJson, err := json.Marshal(newStats)
	if err != nil {
		return errors.Wrap(err, "serialize updated stats")
	}
	c.Set(routineName+":runStats:"+isoDate, updatedStatsJson)

	return nil
}

// addRunToStats adds a new run to the stats.
func addRunToStats(stats RoutineRunStats, durationMs int32, errored bool) RoutineRunStats {
	errorCount := int32(0)
	if errored {
		errorCount = 1
	}
	return mergeStats(stats, RoutineRunStats{
		Since:         time.Now(),
		RunCount:      1,
		ErrorCount:    errorCount,
		MinDurationMs: durationMs,
		AvgDurationMs: durationMs,
		MaxDurationMs: durationMs,
	})
}
