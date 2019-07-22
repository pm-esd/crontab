package crontab

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Crontab 表示cron表的Crontab结构
type Crontab struct {
	ticker *time.Ticker
	jobs   []job
}

// job 在cron表中工作
type job struct {
	second    map[int]struct{}
	min       map[int]struct{}
	hour      map[int]struct{}
	day       map[int]struct{}
	month     map[int]struct{}
	dayOfWeek map[int]struct{}

	fn   interface{}
	args []interface{}
}

// tick 是每分钟发生的单个任务
type tick struct {
	second    int
	min       int
	hour      int
	day       int
	month     int
	dayOfWeek int
}

// New 新的初始化并返回新的cron表
func New() *Crontab {
	return new(time.Minute)
}

// new 创建了新的crontab，arg用于测试目的
func new(t time.Duration) *Crontab {
	c := &Crontab{
		ticker: time.NewTicker(t),
	}

	go func() {
		for t := range c.ticker.C {
			c.runScheduled(t)
		}
	}()

	return c
}

// AddJob 到cron表
func (c *Crontab) AddJob(schedule string, fn interface{}, args ...interface{}) error {
	j, err := parseSchedule(schedule)
	if err != nil {
		return err
	}

	if fn == nil || reflect.ValueOf(fn).Kind() != reflect.Func {
		return fmt.Errorf("Cron 必须是func（）")
	}

	fnType := reflect.TypeOf(fn)
	if len(args) != fnType.NumIn() {
		return fmt.Errorf("func（）参数的数量和提供的参数的数量不匹配")
	}

	for i := 0; i < fnType.NumIn(); i++ {
		a := args[i]
		t1 := fnType.In(i)
		t2 := reflect.TypeOf(a)

		if t1 != t2 {
			if t1.Kind() != reflect.Interface {
				return fmt.Errorf("Param with index %d shold be `%s` not `%s`", i, t1, t2)
			}
			if !t2.Implements(t1) {
				return fmt.Errorf("Param with index %d of type `%s` doesn't implement interface `%s`", i, t2, t1)
			}
		}
	}

	// 全部选中，将作业添加到crontab
	j.fn = fn
	j.args = args
	c.jobs = append(c.jobs, j)
	return nil
}

// MustAddJob 就像AddJob，但如果作业有问题就会发生失败
func (c *Crontab) MustAddJob(schedule string, fn interface{}, args ...interface{}) {
	if err := c.AddJob(schedule, fn, args...); err != nil {
		panic(err)
	}
}

// Shutdown the cron table schedule
func (c *Crontab) Shutdown() {
	c.ticker.Stop()
}

// Clear all jobs from cron table
func (c *Crontab) Clear() {
	c.jobs = []job{}
}

// RunAll jobs in cron table, shcheduled or not
func (c *Crontab) RunAll() {
	for _, j := range c.jobs {
		go j.run()
	}
}

// RunScheduled jobs
func (c *Crontab) runScheduled(t time.Time) {
	tick := getTick(t)
	for _, j := range c.jobs {
		if j.tick(tick) {
			go j.run()
		}
	}
}

// run the job using reflection
func (j job) run() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Crontab error", r)
		}
	}()
	v := reflect.ValueOf(j.fn)
	rargs := make([]reflect.Value, len(j.args))
	for i, a := range j.args {
		rargs[i] = reflect.ValueOf(a)
	}
	v.Call(rargs)
}

// tick decides should the job be lauhcned at the tick
func (j job) tick(t tick) bool {

	if _, ok := j.second[t.second]; !ok {
		return false
	}

	if _, ok := j.min[t.min]; !ok {
		return false
	}

	if _, ok := j.hour[t.hour]; !ok {
		return false
	}

	_, day := j.day[t.day]
	_, dayOfWeek := j.dayOfWeek[t.dayOfWeek]
	if !day && !dayOfWeek {
		return false
	}

	if _, ok := j.month[t.month]; !ok {
		return false
	}

	return true
}

// 用于解析调度字符串的正则表达式
var (
	matchSpaces = regexp.MustCompile("\\s+")
	matchN      = regexp.MustCompile("(.*)/(\\d+)")
	matchRange  = regexp.MustCompile("^(\\d+)-(\\d+)$")
)

// parseSchedule 创建具有填充时间的作业结构以启动，或者如果synthax错误则创建错误
func parseSchedule(s string) (j job, err error) {
	s = matchSpaces.ReplaceAllLiteralString(s, " ")
	parts := strings.Split(s, " ")
	if len(parts) != 6 {
		return job{}, errors.New("Schedule string must have five components like * * * * *")
	}

	j.second, err = parsePart(parts[0], 0, 59)
	if err != nil {
		return j, err
	}

	j.min, err = parsePart(parts[1], 0, 59)
	if err != nil {
		return j, err
	}

	j.hour, err = parsePart(parts[2], 0, 23)
	if err != nil {
		return j, err
	}

	j.day, err = parsePart(parts[3], 1, 31)
	if err != nil {
		return j, err
	}

	j.month, err = parsePart(parts[4], 1, 12)
	if err != nil {
		return j, err
	}

	j.dayOfWeek, err = parsePart(parts[5], 0, 6)
	if err != nil {
		return j, err
	}

	switch {
	case len(j.day) < 31 && len(j.dayOfWeek) == 7:
		j.dayOfWeek = make(map[int]struct{})
	case len(j.dayOfWeek) < 7 && len(j.day) == 31:
		j.day = make(map[int]struct{})
	default:

	}

	return j, nil
}

// parsePart 从日程表字符串中解析单个日程表部分
func parsePart(s string, min, max int) (map[int]struct{}, error) {

	r := make(map[int]struct{}, 0)

	if s == "*" {
		for i := min; i <= max; i++ {
			r[i] = struct{}{}
		}
		return r, nil
	}

	// */2 1-59/5 pattern
	if matches := matchN.FindStringSubmatch(s); matches != nil {
		localMin := min
		localMax := max
		if matches[1] != "" && matches[1] != "*" {
			if rng := matchRange.FindStringSubmatch(matches[1]); rng != nil {
				localMin, _ = strconv.Atoi(rng[1])
				localMax, _ = strconv.Atoi(rng[2])
				if localMin < min || localMax > max {
					return nil, fmt.Errorf("Out of range for %s in %s. %s must be in range %d-%d", rng[1], s, rng[1], min, max)
				}
			} else {
				return nil, fmt.Errorf("Unable to parse %s part in %s", matches[1], s)
			}
		}
		n, _ := strconv.Atoi(matches[2])
		for i := localMin; i <= localMax; i += n {
			r[i] = struct{}{}
		}
		return r, nil
	}

	// 1,2,4  or 1,2,10-15,20,30-45 pattern
	parts := strings.Split(s, ",")
	for _, x := range parts {
		if rng := matchRange.FindStringSubmatch(x); rng != nil {
			localMin, _ := strconv.Atoi(rng[1])
			localMax, _ := strconv.Atoi(rng[2])
			if localMin < min || localMax > max {
				return nil, fmt.Errorf("Out of range for %s in %s. %s must be in range %d-%d", x, s, x, min, max)
			}
			for i := localMin; i <= localMax; i++ {
				r[i] = struct{}{}
			}
		} else if i, err := strconv.Atoi(x); err == nil {
			if i < min || i > max {
				return nil, fmt.Errorf("Out of range for %d in %s. %d must be in range %d-%d", i, s, i, min, max)
			}
			r[i] = struct{}{}
		} else {
			return nil, fmt.Errorf("Unable to parse %s part in %s", x, s)
		}
	}

	if len(r) == 0 {
		return nil, fmt.Errorf("Unable to parse %s", s)
	}

	return r, nil
}

// getTick 从时间返回tick结构
func getTick(t time.Time) tick {
	return tick{
		second:    t.Second(),
		min:       t.Minute(),
		hour:      t.Hour(),
		day:       t.Day(),
		month:     int(t.Month()),
		dayOfWeek: int(t.Weekday()),
	}
}
