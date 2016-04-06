package uwsgi

import (
	"encoding/json"
	"fmt"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
	"net"
	"net/url"
	"strconv"
	"time"
)

var timeout = 5 * time.Second

type Uwsgi struct {
	Servers []string `toml:"server"`
}

func (u *Uwsgi) Description() string {
	return "Read uWSGI metrics."
}

func (u *Uwsgi) SampleConfig() string {
	return `
    ## List with urls of uWSGI Stats servers. Url must match pattern:
    ## scheme://address[:port]
    ##
    ## For example:
    ## servers = ["tcp://localhost:5050", "http://localhost:1717", "unix:///tmp/statsock"]
    servers = []
`
}

func (u *Uwsgi) Gather(acc telegraf.Accumulator) error {
	for _, s := range u.Servers {
		n, err := url.Parse(s)
		if err != nil {
			return fmt.Errorf("Could not parse uWSGI Stats Server url '%s': %s", s, err)
		}

		u.gatherServer(acc, n)

	}
	return nil
}

func (u *Uwsgi) gatherServer(acc telegraf.Accumulator, url *url.URL) error {
	var err error
	var conn net.Conn

	if url.Scheme == "unix" {
		conn, err = net.DialTimeout(url.Scheme, url.Path, timeout)
	} else {
		conn, err = net.DialTimeout(url.Scheme, url.Host, timeout)
	}

	if err != nil {
		return fmt.Errorf("Could not connect to uWSGI Stats Server '%s': %s", url.String(), err)
	}
	defer conn.Close()

	var s StatsServer
	s.Url = url.String()

	dec := json.NewDecoder(conn)
	dec.Decode(&s)

	u.gatherStatServer(acc, &s)
	u.gatherWorkers(acc, &s)

	return nil
}

func (u *Uwsgi) gatherStatServer(acc telegraf.Accumulator, s *StatsServer) error {
	fields := map[string]interface{}{
		"listen_queue":        s.ListenQueue,
		"listen_queue_errors": s.ListenQueueErrors,
		"signal_queue":        s.SignalQueue,
		"load":                s.Load,
	}

	tags := map[string]string{
		"url":     s.Url,
		"pid":     strconv.Itoa(s.Pid),
		"uid":     strconv.Itoa(s.Uid),
		"gid":     strconv.Itoa(s.Gid),
		"version": s.Version,
		"cwd":     s.Cwd,
	}
	acc.AddFields("uwsgi_overview", fields, tags)

	return nil

}

func (u *Uwsgi) gatherWorkers(acc telegraf.Accumulator, s *StatsServer) error {
	for _, w := range s.Workers {
		fields := map[string]interface{}{
			"requests":       w.Requests,
			"accepting":      w.Accepting,
			"delta_request":  w.DeltaRequests,
			"harakiri_count": w.HarakiriCount,
			"signals":        w.Signals,
			"signal_queue":   w.SignalQueue,
			"status":         w.Status,
			"rss":            w.Rss,
			"vsz":            w.Vsz,
			"running_time":   w.RunningTime,
			"last_spawn":     w.LastSpawn,
			"respawn_count":  w.RespawnCount,
			"tx":             w.Tx,
			"avg_rt":         w.AvgRt,
		}
		tags := map[string]string{
			"id":  strconv.Itoa(w.Id),
			"url": s.Url,
			"pid": strconv.Itoa(w.Pid),
		}

		acc.AddFields("uwsgi_workers", fields, tags)
	}

	return nil
}

func init() {
	inputs.Add("uwsgi", func() telegraf.Input { return &Uwsgi{} })
}