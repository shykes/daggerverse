// Record and replay interactive terminal sessions

package main

import (
	"context"
	"strings"
	"math/rand"
	"time"
	"encoding/json"
	"fmt"
)

var (
	asciinemaDigest = "sha256:dc5fed074250b307758362f0b3045eb26de59ca8f6747b1d36f665c1f5dcc7bd"
	aggGitCommit = "84ef0590c9deb61d21469f2669ede31725103173"
	defaultContainer = dag.Wolfi().Container(WolfiContainerOpts{Packages: []string{"dagger"}})
	defaultShell = []string{"/bin/sh"}
	defaultPrompt = "$ "
	defaultWidth = 80
	defaultHeight = 24
)

func New(
	// Terminal width
	// +optional
	width int,
	// Terminal height
	// +optional
	height int,
	// OpenAI auth key, for AI features
	// +optional
	key *Secret,
	// Containerized environment for executing commands
	// +optional
	container *Container,
	// Shell to use when executing commands
	// +optional
	shell []string,
	// The prompt shown by the interactive shell
	// +optional
	prompt string,
) *Termcast {
	if container == nil {
		container = defaultContainer
	}
	if shell == nil {
		shell = defaultShell
	}
	if prompt == "" {
		prompt = defaultPrompt
	}
	if width == 0 {
		width = defaultWidth
	}
	if height == 0 {
		height = defaultHeight
	}
	return &Termcast{
		Height: height,
		Width: width,
		Key: key,
		Container: container,
		Shell: shell,
		Prompt: prompt,
	}
}

type Termcast struct{
	Height int
	Width int
	// +private
	Events []*Event
	// Time elapsed since beginning of the session, in milliseconds
	Clock int
	// +private
	Key *Secret
	// The containerized environment where commands are executed
	// See Exec()
	Container *Container
	// +private
	Shell []string
	// +private
	Prompt string
}

type Event struct {
	Time int // milliseconds
	Code string
	Data string
}

func (e *Event) Encode() (string, error) {
	out, err := json.Marshal([3]interface{}{float64(e.Time) / 1000, e.Code, e.Data})
	return string(out), err
}

func (m *Termcast) Print(data string) *Termcast {
	m.Events = append(m.Events, &Event{
		Time: m.Clock,
		Code: "o",
		Data: strings.Replace(data, "\n", "\r\n", -1),
	})
	return m
}

func (m *Termcast) Append(other *Termcast) *Termcast {
	for _, e := range other.Events {
		 newEvent := &Event{
			Time: m.Clock + e.Time,
			Code: e.Code,
			Data: e.Data,
		}
		m.Events = append(m.Events, newEvent)
		if newEvent.Time > m.Clock {
			m.Clock = newEvent.Time
		}
	}
	return m
}


func (m *Termcast) WaitRandom(min, max int) *Termcast {
	rand.Seed(time.Now().UnixNano())
	return m.Wait(min + rand.Intn(max - min))
}

func (m *Termcast) Wait(
	// wait time, in milliseconds
	ms int,
) *Termcast {
	m.Clock += ms
	return m
}

// Record the execution of a (real) command in an interactive shell
func (m *Termcast) Exec(
	ctx context.Context,
	// The command to execute
	cmd string,
	// How long to wait before showing the command output, in milliseconds
	// +default=100
	delay int,
) (*Termcast, error) {
	m.Container = m.Container.WithExec(m.Shell, ContainerWithExecOpts{
		Stdin: cmd,
		RedirectStdout: "/tmp/output",
		RedirectStderr: "/tmp/output",
		ExperimentalPrivilegedNesting: true, // for dagger-in-dagger
	})
	m = m.
		Print(m.Prompt).
		Keystrokes(cmd).
		Enter().
		Wait(delay)
	output, err := m.Container.File("/tmp/output").Contents(ctx)
	if err != nil {
		return m, err
	}
	return m.Print(output), nil
}

// Simulate a human typing text
func (m *Termcast) Keystrokes(
	// Data to input as keystrokes
	data string,
) *Termcast {
	for _, c := range data {
		m = m.Keystroke(string(c))
	}
	return m
}

// +private
// Simulate a human entering a single keystroke
func (m *Termcast) Keystroke(
	// Data to input as keystrokes
	data string,
) *Termcast {
	return m.WaitRandom(5, 200).Print(data)
}

// Simulate pressing the enter key
func (m *Termcast) Enter() *Termcast {
	return m.Keystroke("\r\n")
}

// Simulate pressing the backspace key
func (m *Termcast) Backspace(
	// Number of backspaces
	// +default=1
	repeat int,
) *Termcast {
	for i := 0; i < repeat; i += 1 {
		m = m.Keystroke("\b \b")
	}
	return m
}

func (m *Termcast) Encode() (string, error) {
	var out strings.Builder
	if err := json.NewEncoder(&out).Encode(map[string]interface{}{
		"version": 2,
		"width": m.Width,
		"height": m.Height,
	}); err != nil {
		return out.String(), err
	}
	for _, e := range m.Events {
		line, err := e.Encode()
		if err != nil {
			return out.String(), err
		}
		out.Write([]byte(line + "\n"))
	}
	return out.String(), nil
}

func (m *Termcast) Castfile() (*File, error) {
	contents, err := m.Encode()
	if err != nil {
		return nil, err
	}
	castfile := dag.
		Directory().
		WithNewFile("castfile", contents).
		File("castfile")
	return castfile, nil
}


func (m *Termcast) Play() (*Terminal, error) {
	castfile, err := m.Castfile()
	if err != nil {
		return nil, err
	}
	term := dag.
		Container().
		From("ghcr.io/asciinema/asciinema@" + asciinemaDigest).
		WithoutEntrypoint().
		WithFile("term.cast", castfile).
		Terminal(ContainerTerminalOpts{
			Cmd: []string{"asciinema", "play", "term.cast"},
		})
	return term, nil
}

func (m *Termcast) Demo() *Termcast {
	var message = `hello daggernauts! this recording is entirely generated by a dagger function :)`
	return m.
		Print("$ ").
		Keystrokes("echo " + message).
		Enter().
		Print(message).Enter().
		Print("$ ").
		Keystrokes("ls -l").Enter().
		WaitRandom(1000, 2000).
		Print(`
total 32
-rw-------@  1 shykes  staff  10280 Feb 15 15:49 LICENSE
drwxr-xr-x@ 11 shykes  staff    352 Feb 21 15:30 containers
drwxr-xr-x   4 shykes  staff    128 Mar  1 17:48 core
drwxr-xr-x@ 16 shykes  staff    512 Mar  6 12:46 dagger
-rw-r--r--   1 shykes  staff    124 Mar  8 13:46 dagger.json
drwxr-xr-x@ 17 shykes  staff    544 Mar  3 21:54 daggy
drwxr-xr-x   9 shykes  staff    288 Feb 21 15:30 datetime
drwxr-xr-x  13 shykes  staff    416 Mar  4 11:28 docker
drwxr-xr-x  13 shykes  staff    416 Mar  2 01:02 docker-compose
drwxr-xr-x@ 11 shykes  staff    352 Mar  7 13:16 dsh
drwxr-xr-x@  3 shykes  staff     96 Feb 21 14:36 go
drwxr-xr-x@  9 shykes  staff    288 Feb 21 15:30 gptscript
drwxr-xr-x  12 shykes  staff    384 Feb 21 15:30 graphiql
drwxr-xr-x@ 11 shykes  staff    352 Feb 21 15:30 hello
drwxr-xr-x   9 shykes  staff    288 Feb 21 15:30 imagemagick
drwxr-xr-x   7 shykes  staff    224 Feb 21 15:30 make
drwxr-xr-x   8 shykes  staff    256 Feb 21 15:30 myip
drwxr-xr-x   9 shykes  staff    288 Jan 25 00:10 ollama
drwxr-xr-x   3 shykes  staff     96 Mar  6 12:38 scratch
drwxr-xr-x@ 12 shykes  staff    384 Feb 21 15:30 slim
drwxr-xr-x@ 11 shykes  staff    352 Mar  6 12:22 supercore
drwxr-xr-x  14 shykes  staff    448 Feb 21 15:30 supergit
drwxr-xr-x@ 11 shykes  staff    352 Feb 21 15:30 tailscale
drwxr-xr-x  14 shykes  staff    448 Mar  9 02:26 termcast
drwxr-xr-x  12 shykes  staff    384 Feb 21 15:30 tmate
drwxr-xr-x   8 shykes  staff    256 Feb 21 15:30 ttlsh
drwxr-xr-x  10 shykes  staff    320 Feb 26 13:09 utils
drwxr-xr-x@ 12 shykes  staff    384 Feb 21 15:30 wolfi
`).
		Wait(1000)
}

func (m *Termcast) Gif() (*File, error) {
	agg := dag.
		Git("https://github.com/asciinema/agg").
		Tag(aggGitCommit).
		// Tag("v1.4.3").
		Tree().
		DockerBuild().
		WithoutEntrypoint()
	castfile, err := m.Castfile()
	if err != nil {
		return nil, err
	}
	gif := agg.
		WithMountedFile("term.cast", castfile).
		WithExec([]string{"agg", "term.cast", "cast.gif"}).
		File("cast.gif")
	return gif, nil
}

func (m *Termcast) Decode(data string) (*Termcast, error) {
	dec := json.NewDecoder(strings.NewReader(data))
	for dec.More() {
		var o [3]interface{}
		if err := dec.Decode(&o); err != nil {
			return nil, err
		}
		seconds, ok := o[0].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid format")
		}
		milliseconds := int(seconds * 1000)
		code, ok := o[1].(string)
		if !ok {
			return nil, fmt.Errorf("invalid format")
		}
		data, ok := o[2].(string)
		if !ok {
			return nil, fmt.Errorf("invalid format")
		}
		e := &Event{
			Time: milliseconds,
			Code: code,
			Data: data,
		}
		m.Events = append(m.Events, e)
		if e.Time > m.Clock {
			m.Clock = e.Time
		}
	}
	return m, nil
}

func (m *Termcast) Imagine(
	ctx context.Context,
	// A description of the terminal session
	// +default="surprise me! an epic interactive session with a shell, language repl or database repl of your choice. the more exotic the better. Not python!"
	prompt string) (*Termcast, error) {
	prompt = `You are a terminal simulator.
- I give you a description of a terminal session
- You give me a stream of asciicast v2 events describing the sesion
- On asciicast event per line
- Don't print the asciicast header
- Don't print anything other than the stream of events
- For special characters, use "\u" not "\x"
- Here is an example of output:
------
[1.000000, "o", "$ "]
[1.500000, "o", "l"]
[1.600000, "o", "s"]
[1.700000, "o", " "]
[1.800000, "o", "-"]
[1.900000, "o", "l"]
[2.000000, "o", "\r\n"]
[2.100000, "o", "total 32\r\n"]
[2.200000, "o", "-rw-r--r--  1 user  staff  1024 Mar  7 10:00 file1.txt\r\n"]
------

Prompt:
` + prompt
	out, err := dag.Daggy().Do(ctx, prompt, DaggyDoOpts{
		Token: m.Key,
	})
	if err != nil {
		return nil, err
	}
	return m.Decode(out)
}
