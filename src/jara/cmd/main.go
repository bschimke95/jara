package main

import (
	"github.com/canonical/k8s/pkg/client/juju"
	"github.com/rivo/tview"
)

func main() {

	client := juju.NewClient()

	modelStatus, err := client.CurrentModel()
	if err != nil {
		panic(err)
	}

	flex := tview.NewFlex()
	applicationList := tview.NewList()

	for _, application := range modelStatus.Applications {
		applicationList.AddItem(application.Name, "", 0, nil)
	}

	flex.SetTitle("Applications")
	flex.SetTitleAlign(tview.AlignLeft)
	flex.SetBorder(true)
	flex.AddItem(applicationList, 0, 1, true)

	if err := tview.NewApplication().SetRoot(flex, true).Run(); err != nil {
		panic(err)
	}
}
