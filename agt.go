package main

import (
	"context"
	"reflect"

	// "flag"
	// 	"fmt"
	"os"

	// "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	// "github.com/aws/aws-sdk-go-v2/service/sso"
	// "github.com/aws/aws-sdk-go-v2/service/ssooidc"

	// "github.com/chromedp/chromedp"

	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
)

func main() {
	// Debugging
	logrus.SetLevel(logrus.DebugLevel)

	// Get credential defaults from shared config
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithSharedConfigProfile("iac-dev"),
	)
	if err != nil {
		logrus.Errorln(err)
		os.Exit(1)
	}
	logrus.Debugf("Config: %+v", cfg)

	providerValueField := reflect.ValueOf(cfg.Credentials).Elem().Field(0)
	providerValue := GetUnexportedField(providerValueField)
	optionsValueField := reflect.ValueOf(providerValue).Elem().Field(0)
	optionsValue := GetUnexportedField(optionsValueField)

	// spew.Dump(cfg.Credentials.Client)
	spew.Dump(reflect.ValueOf(optionsValue))
	spew.Dump(reflect.ValueOf(providerValue))

	// logrus.Debugf("Config SSO Start URL: %+s", )
}
