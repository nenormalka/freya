package types

import (
	"fmt"
	"runtime"
	"strings"

	"golang.org/x/mod/modfile"
)

type (
	Package struct {
		Name    string
		Version string
	}

	AppInfo struct {
		Packages     []Package
		AppVersion   string
		GoVersion    string
		ProtoVersion string
		FreyaVersion string
	}
)

const (
	freyaMask = "freya"
	protoMask = "go-gen-proto"
)

var appInfo = &AppInfo{}

func SetAppInfo(info *AppInfo) {
	appInfo = info
}

func GetAppVersion() string {
	return appInfo.AppVersion
}

func GetGoVersion() string {
	return appInfo.GoVersion
}

func GetProtoVersion() string {
	return appInfo.ProtoVersion
}

func GetFreyaVersion() string {
	return appInfo.FreyaVersion
}

func GetAppInfo(gomod []byte, releaseID, protoPackage string) (*AppInfo, error) {
	if len(gomod) == 0 {
		return &AppInfo{AppVersion: releaseID}, nil
	}

	f, err := modfile.Parse("", gomod, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.mod err %w", err)
	}

	if protoPackage == "" {
		protoPackage = protoMask
	}

	var (
		protoVersion string
		freyaVersion string
	)

	packages := make([]Package, len(f.Require))
	for i := range f.Require {
		packages[i] = Package{
			Name:    f.Require[i].Mod.Path,
			Version: f.Require[i].Mod.Version,
		}

		if strings.Contains(packages[i].Name, protoPackage) {
			protoVersion = packages[i].Version
		}

		if strings.Contains(packages[i].Name, freyaMask) {
			freyaVersion = packages[i].Version
		}
	}

	return &AppInfo{
		Packages:     packages,
		AppVersion:   releaseID,
		GoVersion:    runtime.Version(),
		ProtoVersion: protoVersion,
		FreyaVersion: freyaVersion,
	}, nil
}
