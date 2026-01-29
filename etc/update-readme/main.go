package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/kellegous/poop"
)

type Task struct {
	Title string
	Done  bool
}

func (t *Task) Check() string {
	if t.Done {
		return "x"
	}
	return " "
}

type Section struct {
	Title string
	Task  []Task
}

func (s *Section) Check() string {
	for _, task := range s.Task {
		if !task.Done {
			return " "
		}
	}
	return "x"
}

type Sections []Section

func (s Sections) Status() string {
	if pct := s.Pct(); pct < 1 {
		return "In Progress"
	}
	return "Complete"
}

func (s Sections) Pct() float64 {
	total := 0
	done := 0
	for _, section := range s {
		total += len(section.Task)
		for _, task := range section.Task {
			if task.Done {
				done++
			}
		}
	}
	return float64(done) / float64(total)
}

func (s Sections) ProgressBar(width int) string {
	pct := s.Pct()
	symbols := []string{" ", "▏", "▎", "▍", "▌", "▋", "▊", "▉", "█"}
	numFull := int(pct * float64(width))
	remainder := (pct * float64(width)) - float64(numFull)
	symbolIndex := int(remainder * 8)

	bar := strings.Repeat("█", numFull)
	if numFull < width {
		bar += symbols[symbolIndex]
		bar += strings.Repeat(" ", width-numFull-1)
	}

	return fmt.Sprintf("```\n|%s| %d%%\n```", bar, int(pct*100))
}

var sections = Sections{
	{
		Title: "meshcore.Conn",
		Task: []Task{
			{
				Title: "sendAdvert",
				Done:  true,
			},
			{
				Title: "setAdvertName",
			},
			{
				Title: "setAdvertLatLong",
			},
			{
				Title: "setTxPower",
			},
			{
				Title: "setRadioParams",
			},
			{
				Title: "getContacts",
				Done:  true,
			},
			{
				Title: "sendTextMessage",
				Done:  true,
			},
			{
				Title: "sendChannelTextMessage",
				Done:  true,
			},
			{
				Title: "syncNextMessage",
				Done:  true,
			},
			{
				Title: "getDeviceTime",
				Done:  true,
			},
			{
				Title: "setDeviceTime",
			},
			{
				Title: "importContact",
				Done:  true,
			},
			{
				Title: "exportContact",
				Done:  true,
			},
			{
				Title: "shareContact",
			},
			{
				Title: "removeContact",
				Done:  true,
			},
			{
				Title: "addOrUpdateContact",
				Done:  true,
			},
			{
				Title: "setContactPath",
			},
			{
				Title: "resetPath",
			},
			{
				Title: "reboot",
				Done:  true,
			},
			{
				Title: "getBatteryVoltage",
				Done:  true,
			},
			{
				Title: "deviceQuery",
				Done:  true,
			},
			{
				Title: "exportPrivateKey",
				Done:  true,
			},
			{
				Title: "importPrivateKey",
				Done:  true,
			},
			{
				Title: "login",
			},
			{
				Title: "getStatus",
				Done:  true,
			},
			{
				Title: "getTelemetry",
				Done:  true,
			},
			{
				Title: "sendBinaryRequest",
			},
			{
				Title: "getChannel",
				Done:  true,
			},
			{
				Title: "setChannel",
				Done:  true,
			},
			{
				Title: "deleteChannel",
				Done:  true,
			},
			{
				Title: "getChannels",
				Done:  true,
			},
			{
				Title: "sign",
			},
			{
				Title: "tracePath",
			},
			{
				Title: "setOtherParams",
			},
			{
				Title: "getNeighbors",
			},
		},
	},
	{
		Title: "Transports",
		Task: []Task{
			{
				Title: "BLE",
			},
			{
				Title: "USB/Serial",
			},
		},
	},
}

var readmeTemplate = template.Must(template.New("readme").Parse(`
# Meshcore Companion Radio in Go

A Go module for interacting with a [MeshCore](https://github.com/meshcore-dev/MeshCore) device running the [Companion Radio Firmware](https://github.com/meshcore-dev/MeshCore/blob/main/examples/companion_radio/main.cpp).

## Status

**{{ .Status }}**

{{ .ProgressBar 50 }}

## TODO:

{{ range . }}
 - [{{ .Check }}] {{ .Title }}
 {{ range .Task }}
   - [{{ .Check }}] {{ .Title }}
 {{ end }}
{{ end }}
`))

func main() {
	if err := run(); err != nil {
		poop.HitFan(err)
	}
}

func run() error {
	for _, section := range sections {
		sort.Slice(section.Task, func(i, j int) bool {
			return section.Task[i].Title < section.Task[j].Title
		})
	}

	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Title < sections[j].Title
	})

	if err := readmeTemplate.Execute(os.Stdout, sections); err != nil {
		return poop.Chain(err)
	}
	return nil
}
