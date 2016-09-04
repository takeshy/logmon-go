package main

import (
	"fmt"
	"os"
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
	commandRe := regexp.MustCompile("^[^#].*")
	var path string
	var target, ignore *regexp.Regexp
	for i:=0; i < len(contents); i++ {
		if fileRe.MatchString(contents[i]) {
			if target != nil || ignore != nil {
				panic(strconv.Itoa(i) + ":format error")
			}
			path = fileRe.ReplaceAllString(contents[i], "$1")
		} else if targetRe.MatchString(contents[i]) {
			if path == "" {
				panic(strconv.Itoa(i) + "target format error")
			}
			target = regexp.MustCompile(targetRe.ReplaceAllString(contents[i], "$1"))
		} else if ignoreRe.MatchString(contents[i]) {
			if path == "" {
				panic(strconv.Itoa(i) + "ignore format error")
			}
			ignore = regexp.MustCompile(ignoreRe.ReplaceAllString(contents[i], "$1"))
		} else if commandRe.MatchString(contents[i]) {
			if path == "" || target == nil {
				panic(strconv.Itoa(i) + "command format error")
			}
			ret = append(ret, Watching{path, target,ignore, contents[i]})
			path = ""
			target = nil
			ignore = nil
		}
	}
	return ret
}

func logMonitor(data Watching) {
	c := tail.Watch(data.Path)
	replaceRe :=  regexp.MustCompile("<%%%%>")
	for {
		select {
		case s := <-c:
			if data.Target.MatchString(s) && (data.Ignore == nil || !data.Ignore.MatchString(s)) {
				message := strings.Replace(s, "'", "\\047", -1)
				message = strings.Replace(message, "$", "\\044", -1)
				command := replaceRe.ReplaceAllString(data.Command, message)
				out, err := exec.Command("sh", "-c", command).Output()
				if err != nil {
					panic(err)
				}
				fmt.Println(string(out))
			}
		}
	}
}

func main(){
	conf := flag.String("f", "/etc/logmon/logmon.conf", "config file(Default: /etc/logmon/logmon.conf)")
	flag.Parse()
	data := parseConf(readConf(*conf))
	for i := range data {
		go logMonitor(data[i])
	}
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan,
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGQUIT)
	_ = <-signalChan
}
