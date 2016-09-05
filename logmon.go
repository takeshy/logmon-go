package main

import (
	"os"
	"sync"
	"fmt"
	"time"
	"flag"
	"strings"
	"io/ioutil"
	"strconv"
	"regexp"
	"os/signal"
	"os/exec"
	"syscall"
	"github.com/takeshy/tail"
)

type Watching struct {
	Path string
	Target *regexp.Regexp
	Ignore *regexp.Regexp
	WaitMillisecond int64
	Command string
}

func readConf(path string) string {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	return string(data)
}

func parseConf(contentStr string) []Watching {
	contents := strings.Split(contentStr, "\n")
	ret := []Watching{}
	fileRe :=  regexp.MustCompile("^:(.*)")
	targetRe := regexp.MustCompile("^\\((.*)\\)$")
	ignoreRe := regexp.MustCompile("^\\[(.*)\\]$")
	timeRe := regexp.MustCompile("^{(.*)}$")
	commandRe := regexp.MustCompile("^[^#].*")
	var path string
	var target, ignore *regexp.Regexp
	var waitMillisecond int64
	for i:=0; i < len(contents); i++ {
		if fileRe.MatchString(contents[i]) {
			if target != nil || ignore != nil {
				panic(strconv.Itoa(i) + ":format error")
			}
			path = fileRe.ReplaceAllString(contents[i], "$1")
		} else if targetRe.MatchString(contents[i]) {
			if path == "" {
				panic(strconv.Itoa(i) + " target appear before path")
			}
			target = regexp.MustCompile(targetRe.ReplaceAllString(contents[i], "$1"))
		} else if ignoreRe.MatchString(contents[i]) {
			if path == "" {
				panic(strconv.Itoa(i) + "ignore appear before path ")
			}
			ignore = regexp.MustCompile(ignoreRe.ReplaceAllString(contents[i], "$1"))
		} else if timeRe.MatchString(contents[i]) {
			if path == "" {
				panic(strconv.Itoa(i) + "time appear before path")
			}
			waitMillisecondStr := timeRe.ReplaceAllString(contents[i], "$1")
			milliSec, err := strconv.ParseInt(waitMillisecondStr, 10, 64)
			if err != nil {
				panic(err)
			}
			waitMillisecond = milliSec
		} else if commandRe.MatchString(contents[i]) {
			if path == "" || target == nil {
				panic(strconv.Itoa(i) + "command format error")
			}
			ret = append(ret, Watching{path, target,ignore, waitMillisecond, contents[i]})
			path = ""
			waitMillisecond = 0
			target = nil
			ignore = nil
		}
	}
	return ret
}

func escapeShell(s string) string {
	message := strings.Replace(s, "'", "\\047", -1)
	message = strings.Replace(message, "$", "\\044", -1)
	return message
}

func executeCommand(conf Watching, targetMessage string){
	replaceRe :=  regexp.MustCompile("<%%%%>")
	command := replaceRe.ReplaceAllString(conf.Command, targetMessage)
	_, err := exec.Command("sh", "-c", command).Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}

func logMonitor(conf Watching) {
	c := tail.Watch(conf.Path)
	var targetMessage string
	var mutex sync.Mutex
	for {
		select {
		case s := <-c:
			if targetMessage != "" {
				mutex.Lock()
				if(targetMessage != ""){
					targetMessage += ("\n" + escapeShell(s))
				}
				mutex.Unlock()
			} else if conf.Target.MatchString(s) && (conf.Ignore == nil || !conf.Ignore.MatchString(s)) {
				targetMessage += escapeShell(s)
				if(conf.WaitMillisecond == 0){
					executeCommand(conf, targetMessage)
					targetMessage = ""
				} else {
					timer := time.NewTimer(time.Duration(conf.WaitMillisecond) * time.Millisecond)
					go func() {
						<-timer.C
						mutex.Lock()
						executeCommand(conf, targetMessage)
						targetMessage = ""
						mutex.Unlock()
					}()
				}
			}
		}
	}
}

func main(){
	conf := flag.String("f", "/etc/logmon/logmon.conf", "config file(Default: /etc/logmon/logmon.conf)")
	flag.Parse()
	confs := parseConf(readConf(*conf))
	for i := range confs {
		go logMonitor(confs[i])
	}
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan,
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGQUIT)
	_ = <-signalChan
}
