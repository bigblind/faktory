package tester

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/mperham/worq"
	"github.com/mperham/worq/cli"
	"github.com/mperham/worq/util"
)

func TestSystem(t *testing.T) {
	cli.SetupLogging(os.Stdout)
	opts := cli.ParseArguments()

	s := worq.NewServer(&worq.ServerOptions{Binding: opts.Binding, StoragePath: "./system.db"})

	util.LogDebug = true
	util.LogInfo = true

	go stacks()
	go cli.HandleSignals(s)
	go pushAndPop()
	go pushAndPop()
	go pushAndPop()

	err := s.Start()
	if err != nil {
		fmt.Println(err)
		return
	}
}

func pushAndPop() {
	defer os.Exit(0)

	client, err := worq.Dial(&worq.ClientOptions{Pwd: "123456"})
	if err != nil {
		handleError(err)
		return
	}
	defer client.Close()

	util.Debug("Pushing")
	for i := 0; i < 10000; i++ {
		if err = pushJob(client, i); err != nil {
			handleError(err)
			return
		}
	}
	util.Debug("Popping")

	for i := 0; i < 10000; i++ {
		job, err := client.Pop("default")
		if err != nil {
			handleError(err)
			return
		}
		if i%100 == 99 {
			err = client.Fail(job.Jid, errors.New("oops"), nil)
		} else {
			err = client.Ack(job.Jid)
		}
		if err != nil {
			handleError(err)
			return
		}
	}
}

func pushJob(client *worq.Client, idx int) error {
	j := &worq.Job{
		Jid:   util.RandomJid(),
		Queue: "default",
		Type:  "SomeJob",
		Args:  []interface{}{1, "string", 3},
	}
	return client.Push(j)
}

func stacks() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGQUIT)
	buf := make([]byte, 1<<20)
	for {
		<-sigs
		stacklen := runtime.Stack(buf, true)
		log.Printf("=== received SIGQUIT ===\n*** goroutine dump...\n%s\n*** end\n", buf[:stacklen])
	}
}

func handleError(err error) {
	fmt.Println(strings.Replace(err.Error(), "\n", "", -1))
}
