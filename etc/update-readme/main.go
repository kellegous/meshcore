package main

import (
	"os"
	"sort"
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

var sections = []Section{
	{
		Title: "meshcore.Conn",
		Task: []Task{
			{
				Title: "sendAdvert",
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
			},
			{
				Title: "sendChannelTextMessage",
			},
			{
				Title: "syncNextMessage",
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
			},
			{
				Title: "exportContact",
			},
			{
				Title: "shareContact",
			},
			{
				Title: "removeContact",
			},
			{
				Title: "addOrUpdateContact",
			},
			{
				Title: "setContactPath",
			},
			{
				Title: "resetPath",
			},
			{
				Title: "reboot",
			},
			{
				Title: "getBatteryVoltage",
				Done:  true,
			},
			{
				Title: "deviceQuery",
			},
			{
				Title: "exportPrivateKey",
			},
			{
				Title: "importPrivateKey",
			},
			{
				Title: "login",
			},
			{
				Title: "getStatus",
			},
			{
				Title: "getTelemetry",
			},
			{
				Title: "sendBinaryRequest",
			},
			{
				Title: "getChannel",
			},
			{
				Title: "setChannel",
			},
			{
				Title: "deleteChannel",
			},
			{
				Title: "getChannels",
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
}

var readmeTemplate = template.Must(template.New("readme").Parse(`
# Meshcore Companion Radio in Go

A Go module for interacting with a [MeshCore](https://github.com/meshcore-dev/MeshCore) device running the [Companion Radio Firmware](https://github.com/meshcore-dev/MeshCore/blob/main/examples/companion_radio/main.cpp).

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
