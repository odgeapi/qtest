package main

import (
        "fmt"
        "github.com/mailgun/mailgun-go"
        "os"
        "os/exec"
        "strconv"
        "strings"
        "time"
)

var files []string = []string{"front.log", "demand.log", "search.log", "crawl.log"}
var covered []int64 = []int64{0, 0, 0, 0}
var uppLines int = 3

// Panics is a handler to grep for panics in the log files (for automating monitoring)
func Panics() []string {
        arg := []string{"panic", "-s"} //start list of fgrep args
        for i, f := range files {
                s, err := os.Stat("log/" + f)
                if err == nil && covered[i] > 0 && s.Size()-covered[i] < 100000000 {
                        if s.Size()-covered[i] == 0 {
                                continue
                        }

                        len := (s.Size()-covered[i])/1024 + 2
                        skip := covered[i] / 1024
                        cmd := exec.Command("/bin/dd",
                                "if=log/"+f,
                                "of=/tmp/"+f,
                                fmt.Sprintf("count=%d", len),
                                fmt.Sprintf("skip=%d", skip),
                                "bs=1024")
                        _, err := cmd.CombinedOutput()
                        if err != nil {
                                fmt.Println("dd:", err)
                                arg = append(arg, "log/"+f)
                        } else {
                                arg = append(arg, "/tmp/"+f)
                        }
                } else {
                        arg = append(arg, "log/"+f)
                }
                if err == nil {
                        covered[i] = s.Size()
                }
        }
        cmd := exec.Command("/bin/touch", arg[2:]...)
        output, err := cmd.CombinedOutput()
       if err != nil {
                fmt.Println("touch error on", arg[2:])
        }
        cmd = exec.Command("/bin/fgrep", arg...)
        output, err = cmd.CombinedOutput()
        if err != nil && err.Error() == "exit status 1" {
                err = nil
        }
        if err != nil {
                fmt.Println(err, "on")
                fmt.Println("/bin/fgrep", arg)
                return []string{"error in reading logs: ", err.Error(), string(output)}
        }
        panics := strings.Split(string(output), "\n")
        if panics[len(panics)-1] == "" { // should end empty newline
                panics = panics[0 : len(panics)-1]
        }
        if len(panics) != 0 {
                fmt.Println("found panics:", len(panics))
                /* for i, v := range panics {
                        fmt.Printf("%d\t->%s<-\n", i, v)
                }*/
        } else {
                panics = nil
        }
        // also check if up
        cmd = exec.Command("upp")
        output, err = cmd.CombinedOutput()
        if err != nil {
                fmt.Println("Error with upp!", err)
        }
        procs := strings.Split(string(output), "\n")
        if procs[len(procs)-1] == "" { // should end empty newline
                procs = procs[0 : len(procs)-1]
        }
        if len(procs) != uppLines {
                fmt.Println("not right # lines from upp:", len(procs))
                return append(panics, procs...)
        } else {
                return panics
        }
}

func removeDupes(a []string, b []string) []string {
        if len(b) == 0 {
                return nil
        }
        if len(a) == 0 {
                return b
        }

        c := make([]string, 0) //iefficient but who cares for this
        for _, v := range b {
                new := true
                for _, av := range a {
                        if v == av {
                                new = false
                                break
                        }
                }
                if new {
                        c = append(c, v)
                }
        }
        return c
}

