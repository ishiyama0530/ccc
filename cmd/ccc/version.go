package main

import (
	"runtime/debug"
	"strings"
)

var readBuildInfo = debug.ReadBuildInfo

func currentVersion() string {
	return resolveVersion(version, readBuildInfo)
}

type buildInfoReader func() (*debug.BuildInfo, bool)

func resolveVersion(embedded string, reader buildInfoReader) string {
	if embedded != "" && embedded != "dev" {
		return embedded
	}

	if reader == nil {
		return embedded
	}

	info, ok := reader()
	if !ok || info == nil {
		return embedded
	}

	if buildVersion := strings.TrimSpace(info.Main.Version); buildVersion != "" && buildVersion != "(devel)" {
		return buildVersion
	}

	if revision := vcsRevision(info.Settings); revision != "" {
		return revision
	}

	return embedded
}

func vcsRevision(settings []debug.BuildSetting) string {
	var revision string
	dirty := false

	for _, setting := range settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			dirty = setting.Value == "true"
		}
	}

	if revision == "" {
		return ""
	}

	if len(revision) > 12 {
		revision = revision[:12]
	}

	if dirty {
		return revision + "-dirty"
	}

	return revision
}
