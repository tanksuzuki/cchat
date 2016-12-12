package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"golang.org/x/crypto/ssh"
)

type Message struct {
	ID    string `json:"id"`
	Date  string `json:"date"`
	Owner string `json:"owner"`
	Body  string `json:"body"`
}

type Messages []Message

func (m *Messages) Add(msg Message) {
	*m = append(*m, msg)
}

func (m Messages) Len() int {
	return len(m)
}

func (m Messages) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m Messages) Less(i, j int) bool {
	return m[i].Date > m[j].Date
}

var (
	opts struct {
		Help     bool   `short:"h" long:"help" description:"print usage"`
		Identity string `short:"i" long:"identity" description:"path to private key file"`
		Port     int    `short:"p" long:"port" description:"ssh destination port" default:"22"`
		Listen   int    `short:"l" long:"listen" description:"web/api listening port" default:"8080"`
	}
	sshconf    ssh.ClientConfig
	remotehost string
)

func main() {
	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	parser := flags.NewParser(&opts, flags.PassDoubleDash)
	args, err := parser.Parse()
	switch {
	case err != nil:
		log.Fatal(err)
	case opts.Help, len(args) == 0:
		parser.Usage = "[OPTIONS] USER@REMOTE_HOST"
		parser.WriteHelp(os.Stdout)
		fmt.Printf("\n")
		os.Exit(0)
	case len(args) > 1:
		log.Fatal("too many arguments given")
	}

	user, host := parseRemoteHost(args[0])
	sshconf.User = user
	sshconf.Auth = []ssh.AuthMethod{getPublicKeyFile(opts.Identity)}
	remotehost = host

	conn, err := ssh.Dial("tcp", remotehost+fmt.Sprintf(":%d", opts.Port), &sshconf)
	if err != nil {
		log.Fatal("failed to dial: " + err.Error())
	}
	conn.Close()

	e.Static("/", "public")
	e.POST("/api/message", postMessage)
	e.GET("/api/messages", getMessages)

	log.Fatal(e.Start("localhost:" + fmt.Sprintf("%d", opts.Listen)))
}

func postMessage(c echo.Context) error {
	if c.FormValue("body") == "" {
		return echo.NewHTTPError(400)
	}

	message := Message{
		Date:  time.Now().UTC().Format(time.RFC3339),
		Owner: sshconf.User,
		Body:  c.FormValue("body"),
	}
	message.ID = genSha1Hash(message.Date + message.Owner + message.Body)

	jsonBytes, err := json.Marshal(message)
	if err != nil {
		log.Print("failed to marshal json: " + err.Error())
		panic(err)
	}

	conn, err := ssh.Dial("tcp", remotehost+fmt.Sprintf(":%d", opts.Port), &sshconf)
	if err != nil {
		log.Fatal("failed to dial: " + err.Error())
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		log.Print("failed to create session: " + err.Error())
		panic(err)
	}
	defer session.Close()

	if err := session.Run("send log " + encodeMessage(string(jsonBytes))); err != nil {
		log.Print("failed to send message")
		panic(err)
	}

	return c.NoContent(204)
}

func getMessages(c echo.Context) error {
	conn, err := ssh.Dial("tcp", remotehost+fmt.Sprintf(":%d", opts.Port), &sshconf)
	if err != nil {
		log.Fatal("failed to dial: " + err.Error())
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		log.Print("failed to create session: " + err.Error())
		panic(err)
	}
	defer session.Close()

	in, err := session.StdinPipe()
	if err != nil {
		panic(err)
	}
	out, err := session.StdoutPipe()
	if err != nil {
		panic(err)
	}

	if err := session.Shell(); err != nil {
		log.Print("failed to start shell: %s", err)
		panic(err)
	}
	fmt.Fprint(in, "terminal length 0\n")
	fmt.Fprint(in, "sh log | inc %SYS-[0-7]-USERLOG_\n")
	fmt.Fprint(in, "exit\n")

	messagelogs := []string{}
	r := regexp.MustCompile("%SYS-[0-7]-USERLOG_(DEBUG|INFO|NOTICE|WARNING|ERR|CRIT|ALERT|EMERG): Message from \\w+\\(user id: .+\\): .*$")
	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		t := scanner.Text()
		if r.MatchString(t) {
			messagelogs = append(messagelogs, t)
		}
		if t == "Press RETURN to get started." {
			break
		}
	}

	var messages Messages
	for _, messagelog := range messagelogs {
		jsonString, err := decodeMessage(getLogBody(messagelog))
		if err != nil {
			log.Print("failed to decode json: " + err.Error())
			continue
		}
		var message Message
		if err := json.Unmarshal([]byte(jsonString), &message); err != nil {
			log.Print("unmarshal err: " + err.Error())
			continue
		}
		messages.Add(message)
	}

	if len(messages) == 0 {
		return echo.NewHTTPError(404)
	}

	sort.Sort(messages)

	return c.JSON(200, messages)
}

func parseRemoteHost(remote string) (user, hostname string) {
	split := strings.Split(remote, "@")
	user = strings.Join(split[0:len(split)-1], "@")
	hostname = split[len(split)-1]
	return
}

func getPublicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

func genSha1Hash(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func encodeMessage(msg string) string {
	return base64.StdEncoding.EncodeToString([]byte(msg))
}

func decodeMessage(msg string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func getLogBody(s string) string {
	split := strings.Split(s, ": ")
	return split[len(split)-1]
}
