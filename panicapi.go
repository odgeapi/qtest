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

func alert(label string, msg []string) {
        if msg == nil {
                return
        }

        _, err := os.Stat("/usr/sbin/sendmail")
        if err != nil {
                fmt.Println("Sending via mailgun as ", err)
                mg := mailgun.NewMailgun("odge.io", "key-0...",
                        "pubkey-...")
                m := mg.NewMessage(
                        "Alerts <email@example.com>",
                        "Panic found or process missing on "+label,
                        strings.Join(msg, "\n"),
                        "backupemail@example.com",
                )
                _, id, err := mg.Send(m)
                if err != nil {
                        fmt.Println("Problem sending", id, err)
                        fmt.Println(strings.Join(msg, "\n"))
                }
        } else {
                fmt.Println("Sending via sendmail")
                cmd := exec.Command("/usr/sbin/sendmail", "email@example.com")
                cmd.Stdin = strings.NewReader("Subject: Panic found or process missing on " + label +
                        "\n\n" + strings.Join(msg, "\n"))
                _, err := cmd.CombinedOutput()
                if err != nil {
                        fmt.Println("Sendmail error", err)
                }
        }
}


func main() {
        label := ""
        if len(os.Args) >= 2 {
                label = os.Args[1]
        }
        if len(os.Args) >= 3 {
                x, err := strconv.Atoi(os.Args[2])
                if err == nil {
                        uppLines = x
                }
        }

        cmd := exec.Command("/usr/bin/touch", "log/front.log", "log/demand.log", "log/search.log", "log/crawl.log")
        cmd.CombinedOutput()
        last := Panics() // get initial set and dont alert
        for {
                new := Panics()
                if len(new) > 0 {
                        uniques := removeDupes(last[:], new[:])
                        //uniques := new // with the new "covered" thing, not needed  think
                        if len(uniques) > 0 {
                                time.Sleep(5 * time.Second)
                                alert(label, uniques)
                        }
                        last = new
                }
                time.Sleep(60 * time.Second)
        }
        fmt.Println(len(last)) // shut it compiler
}
