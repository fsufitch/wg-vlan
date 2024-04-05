package main

import (
	"errors"
	"flag"
	"fmt"
	"slices"
	"strings"

	"github.com/urfave/cli/v2"
)

type ChoicesFlag struct {
	cli.StringFlag
	Choices    []string
	innerValue *choicesFlagValue
}

type choicesFlagValue struct {
	choice     string
	nonDefault bool
	flag       *ChoicesFlag
}

func (cfv choicesFlagValue) String() string {
	return cfv.choice
}

func (cfv choicesFlagValue) Get() string {
	return cfv.choice
}

func (cfv *choicesFlagValue) Set(newValue string) error {
	if cfv.nonDefault {
		return errors.New("cannot specify multiple times")
	}
	if !slices.Contains(cfv.flag.Choices, newValue) {
		return errors.New("invalid choice")
	}
	cfv.nonDefault = true
	cfv.choice = newValue
	if cfv.flag.Destination != nil {
		*cfv.flag.Destination = newValue
	}
	return nil
}

func (cf *ChoicesFlag) Apply(flagSet *flag.FlagSet) error {
	cf.innerValue = &choicesFlagValue{
		flag:   cf,
		choice: cf.Value,
	}
	cf.HasBeenSet = cf.Value != ""
	if cf.Destination != nil {
		*cf.Destination = cf.Value
	}

	cf.Usage = fmt.Sprintf("%s [%s]", cf.Usage, strings.Join(cf.Choices, ", "))

	for _, name := range cf.Names() {
		flagSet.Var(cf.innerValue, name, cf.Usage)
	}
	return nil
}
