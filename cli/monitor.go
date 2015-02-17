package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/agtorre/gocolorize"
	"github.com/kr/beanstalk"
	"github.com/mcuadros/termtable"
)

const (
	HighSeverity = iota
	NormalSeverity
	LowSeverity
)

var TubeStatsRetrievalError = errors.New("Unable to retrieve tube stats")
var HighSeverityStyle = gocolorize.NewColor("white:red")
var NormalSeverityStyle = gocolorize.NewColor("green")

type Monitor struct {
	Host string `short:"h" long:"host" description:"beanstalkd host addr." required:"true" default:"localhost:11300"`

	conn *beanstalk.Conn
}

type TubeStats struct {
	JobsBuried   int
	JobsDelayed  int
	JobsReady    int
	JobsReserved int
	JobsUrgent   int
	Waiting      int
	TotalJobs    int
}

func (m *Monitor) Execute(args []string) error {
	if err := m.Init(); err != nil {
		return err
	}

	if err := m.PrintStats(); err != nil {
		return err
	}

	return nil
}

func (m *Monitor) Init() error {
	var err error
	m.conn, err = beanstalk.Dial("tcp", m.Host)
	if err != nil {
		return err
	}

	return nil
}

func (m *Monitor) PrintStats() error {
	stats, err := m.GetStats()
	if err != nil {
		return err
	}

	table := termtable.NewTable(nil, &termtable.TableOptions{
		Padding:      1,
		UseSeparator: true,
	})

	table.SetHeader([]string{
		"Name", "Buried", "Delayed", "Ready", "Reserved", "Urgent", "Waiting", "Total",
	})

	for t, s := range stats {
		table.AddRow(m.buildLineFromTubeStats(t, s))
	}

	fmt.Println(table.Render())
	return nil
}

func (m *Monitor) buildLineFromTubeStats(name string, s *TubeStats) []string {
	l := make([]string, 0)

	l = append(l, name)
	l = append(l, addStyle(s.JobsBuried, 10, HighSeverity))
	l = append(l, addStyle(s.JobsDelayed, 10, NormalSeverity))
	l = append(l, addStyle(s.JobsReady, 10, NormalSeverity))
	l = append(l, addStyle(s.JobsReserved, 10, NormalSeverity))
	l = append(l, addStyle(s.JobsUrgent, 10, NormalSeverity))
	l = append(l, addStyle(s.Waiting, 10, LowSeverity))
	l = append(l, addStyle(s.TotalJobs, 10, LowSeverity))

	return l
}

func addStyle(i int, l int, severity int) string {
	value := strconv.Itoa(i)
	needs := l - len(value)
	if needs <= 0 {
		return value
	}

	padded := value + strings.Repeat(" ", needs)
	if i > 0 {
		switch severity {
		case HighSeverity:
			return HighSeverityStyle.Paint(padded)
		case NormalSeverity:
			return NormalSeverityStyle.Paint(padded)
		}
	}

	return padded
}

func (m *Monitor) GetStats() (map[string]*TubeStats, error) {
	tubes, err := m.conn.ListTubes()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]*TubeStats, 0)
	for _, tube := range tubes {
		s, err := m.GetStatsForTube(tube)
		if err != nil {
			return nil, err
		}

		stats[tube] = s
	}

	return stats, nil
}

func (m *Monitor) GetStatsForTube(tube string) (*TubeStats, error) {
	t := &beanstalk.Tube{m.conn, tube}
	s, err := t.Stats()
	if err != nil {
		return nil, err
	}

	if name, ok := s["name"]; !ok || name != tube {
		return nil, TubeStatsRetrievalError
	}

	return &TubeStats{
		JobsBuried:   mustConvertToInt(s["current-jobs-buried"]),
		JobsReady:    mustConvertToInt(s["current-jobs-ready"]),
		JobsDelayed:  mustConvertToInt(s["current-jobs-delayed"]),
		JobsReserved: mustConvertToInt(s["current-jobs-reserved"]),
		JobsUrgent:   mustConvertToInt(s["current-jobs-urgent"]),
		Waiting:      mustConvertToInt(s["current-waiting"]),
		TotalJobs:    mustConvertToInt(s["total-jobs"]),
	}, nil
}

func mustConvertToInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}

	return i
}
