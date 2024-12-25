package task

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/atomic"
)

var (
	c       *cron.Cron
	started atomic.Bool
	stopped atomic.Bool
)

func init() {
	started.Store(false)
	stopped.Store(true)
	c = cron.New(cron.WithLogger(cron.DefaultLogger),
		cron.WithSeconds(),
		cron.WithLocation(time.Local),
	)

	fmt.Println("cron init success")
	go func() {
		for range time.NewTicker(3 * time.Second).C {
			entries := c.Entries()
			fmt.Println("entries ", len(entries))
			if len(entries) > 0 {
				if !started.Load() {
					fmt.Println(">>> cron start")
					c.Start()
					started.Store(true)
					stopped.Store(false)
				}
			} else {
				if !stopped.Load() {
					c.Stop()
				}
			}
		}
	}()
}

func AddTask(f func(), spec string) (cron.EntryID, error) {
	entryID, err := c.AddFunc(spec, f)
	if err != nil {
		return cron.EntryID(-1), err
	}
	return entryID, nil
}

func RemoveTask(entryId cron.EntryID) bool {
	c.Remove(entryId)
	return true
}

func Close() {
	entries := c.Entries()
	for _, entry := range entries {
		c.Remove(entry.ID)
	}
	c.Stop()
	stopped.Store(true)
	started.Store(false)
}
