package gen

import (
	"github.com/TIBCOSoftware/flogo-cli/util"
)

const (
	fileTriggerDescriptor string = "trigger.json"
	fileTriggerGo         string = "trigger.go"
	fileTriggerGoTest     string = "trigger_test.go"
)

type TriggerGenerator struct {
}

func (g *TriggerGenerator) Description() string {
	return "generates a trigger project"
}

func (g *TriggerGenerator) Generate(basePath string, data interface{}) error {

	err := fgutil.CreateFileFromTemplate(basePath, fileTriggerDescriptor, tplTriggerDescriptor, data)
	if err != nil {
		return err
	}
	err = fgutil.CreateFileFromTemplate(basePath, fileTriggerGo, tplTriggerGo, data)
	if err != nil {
		return err
	}

	err = fgutil.CreateFileFromTemplate(basePath, fileTriggerGoTest, tplTriggerGoTestGo, data)
	if err != nil {
		return err
	}

	return nil
}

var tplTriggerDescriptor = `{
  "name": "{{.Name}}",
  "version": "0.0.1",
  "type": "flogo:trigger",
  "description": "trigger description",
  "author": "Your Name <you.name@example.org>",
  "settings":[
    {
      "name": "setting",
      "type": "string",
      "value": "default"
    }
  ],
  "output": [
    {
      "name": "output",
      "type": "string"
    }
  ],
  "handler": {
    "settings": [
      {
        "name": "handler_setting",
        "type": "string"
      }
    ]
  }
}`

var tplTriggerGo = `package {{.Name}}

import (
	"github.com/TIBCOSoftware/flogo-lib/core/action"
	"github.com/TIBCOSoftware/flogo-lib/core/trigger"
)

// MyTriggerFactory My Trigger factory
type MyTriggerFactory struct{
	metadata *trigger.Metadata
}

// NewFactory create a new Trigger factory
func NewFactory(md *trigger.Metadata) trigger.Factory {
	return &MyTriggerFactory{metadata:md}
}

// New Creates a new trigger instance for a given id
func (t *MyTriggerFactory) New(config *trigger.Config) trigger.Trigger {
	return &MyTrigger{metadata: t.metadata, config:config}
}

// MyTrigger is a stub for your Trigger implementation
type MyTrigger struct {
	metadata *trigger.Metadata
	config   *trigger.Config
}

// Initialize implements trigger.Init.Initialize
func (t *MyTrigger) Initialize(ctx trigger.InitContext) error {
	return nil
}

// Metadata implements trigger.Trigger.Metadata
func (t *MyTrigger) Metadata() *trigger.Metadata {
	return t.metadata
}

// Start implements trigger.Trigger.Start
func (t *MyTrigger) Start() error {
	// start the trigger
	return nil
}

// Stop implements trigger.Trigger.Start
func (t *MyTrigger) Stop() error {
	// stop the trigger
	return nil
}
`

var tplTriggerGoTestGo = `package {{.Name}}

import (
	"context"
	"io/ioutil"
	"encoding/json"
	"testing"

	"github.com/TIBCOSoftware/flogo-lib/core/action"
	"github.com/TIBCOSoftware/flogo-lib/core/trigger"
)

func getJsonMetadata() string{
	jsonMetadataBytes, err := ioutil.ReadFile("trigger.json")
	if err != nil{
		panic("No Json Metadata found for trigger.json path")
	}
	return string(jsonMetadataBytes)
}

type TestRunner struct {
}

// Execute implements action.Runner.Execute
func (runner *TestRunner) Execute(ctx context.Context, act action.Action, inputs map[string]*data.Attribute) (results map[string]*data.Attribute, err error) {
	return nil, nil
}

const testConfig string = ` + "`" + `{
  "id": "mytrigger",
  "settings": {
    "setting": "somevalue"
  },
  "handlers": [
    {
      "actionId": "test_action",
      "settings": {
        "handler_setting": "somevalue"
      }
    }
  ]
}` + "`" + `

func TestInit(t *testing.T) {

	// New factory
	md := trigger.NewMetadata(getJsonMetadata())
	f := NewFactory(md)

	// New Trigger
	config := trigger.Config{}
	json.Unmarshal([]byte(testConfig), config)
	tgr := f.New(&config)

	runner := &TestRunner{}

	initCtx := &struct {
		handlers []*trigger.Handler
	}{
		handlers: make([]*trigger.Handler, 0, len(config.Handlers))
	}
	tgr.Initialize(initCtx)
}
`
