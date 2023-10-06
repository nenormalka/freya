package metadata

import (
	"context"
	"errors"
	"fmt"
	"math"

	v "github.com/hashicorp/go-version"
	"google.golang.org/grpc/metadata"
)

type (
	PlatformType string
	Ratio        int

	MetadataError    error
	VersionDataError error

	VersionData map[PlatformType]string

	Platform interface {
		IsValid() bool
		IsMobile() bool
		IsIos() bool
		IsAndroid() bool
		IsWeb() bool
		IsMobileWeb() bool
		IsDesktopWeb() bool
		Platform() string
	}
)

const (
	PlatformTypeIOS         PlatformType = "ios"
	PlatformTypeANDROID     PlatformType = "android"
	PlatformTypeWEB         PlatformType = "web"
	PlatformTypeWEBMobile   PlatformType = "web-mobile"
	PlatformTypeWEBMobile2  PlatformType = "web_mobile"
	PlatformTypeDesktop     PlatformType = "desktop"
	PlatformTypeWebDesktop  PlatformType = "web-desktop"
	PlatformTypeWebDesktop2 PlatformType = "web_desktop"
)

const (
	AppInfoFieldAppVersion = "app_version"
	// AppInfoFieldPlatformOSVersion версия операционной системы для mobile, для веб название браузера и его версия
	AppInfoFieldPlatformOSVersion = "platform_os_version"
	AppInfoFieldBuild             = "build"
	AppInfoFieldPlatform          = "platform"
)

const (
	RatioEqual Ratio = 0
	RatioGreat Ratio = 1
	RatioLess  Ratio = -1
	RatioError Ratio = math.MaxInt
)

const (
	featurePrefix = "feature-toggle-%d"
	enabledValue  = "enabled"
)

var (
	ErrEmptyMetadata    MetadataError = errors.New("empty metadata in context")
	ErrKeyNotFound      MetadataError = errors.New("not found key")
	ErrEmptyValuesInKey MetadataError = errors.New("empty values in key")

	ErrEmptyCondition   VersionDataError = errors.New("empty condition for platform")
	ErrConditionVersion VersionDataError = errors.New("invalid condition version")
	ErrContextVersion   VersionDataError = errors.New("invalid context version")
)

func GetAppVersion(ctx context.Context) (string, error) {
	return GetDataFromCtx(ctx, AppInfoFieldAppVersion)
}

func GetPlatform(ctx context.Context) (string, error) {
	return GetDataFromCtx(ctx, AppInfoFieldPlatform)
}

func GetPlatformType(ctx context.Context) (Platform, error) {
	platform, err := GetDataFromCtx(ctx, AppInfoFieldPlatform)
	return PlatformType(platform), err
}

func GetPlatformOS(ctx context.Context) (string, error) {
	return GetDataFromCtx(ctx, AppInfoFieldPlatformOSVersion)
}

func GetBuild(ctx context.Context) (string, error) {
	return GetDataFromCtx(ctx, AppInfoFieldBuild)
}

func GetCustomKey(ctx context.Context, key string) (string, error) {
	return GetDataFromCtx(ctx, key)
}

func CompareVersions(ctx context.Context, vd VersionData) (Ratio, error) {
	platformMD, err := GetPlatform(ctx)
	if err != nil {
		return RatioError, err
	}

	version, ok := vd[PlatformType(platformMD)]
	if !ok {
		return RatioError, ErrEmptyCondition
	}

	conditionVersion, err := v.NewSemver(version)
	if err != nil {
		return RatioError, ErrConditionVersion
	}

	versionMD, err := GetAppVersion(ctx)
	if err != nil {
		return RatioError, err
	}

	contextVersion, err := v.NewSemver(versionMD)
	if err != nil {
		return RatioError, ErrContextVersion
	}

	return Ratio(contextVersion.Compare(conditionVersion)), nil
}

func GetDataFromCtx(ctx context.Context, key string) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", ErrEmptyMetadata
	}

	dates, ok := md[key]
	if !ok {
		return "", ErrKeyNotFound
	}

	if len(dates) == 0 {
		return "", ErrEmptyValuesInKey
	}

	return dates[0], nil
}

func FeatureToggleIsEnabled(ctx context.Context, toggle int) bool {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}

	dates, ok := md[fmt.Sprintf(featurePrefix, toggle)]
	if !ok {
		return false
	}

	if len(dates) == 0 {
		return false
	}

	return dates[0] == enabledValue
}

func (p PlatformType) Platform() string {
	return string(p)
}

func (p PlatformType) IsValid() bool {
	return p.IsMobile() || p.IsWeb()
}

func (p PlatformType) IsMobile() bool {
	return p.IsIos() || p.IsAndroid()
}

func (p PlatformType) IsWeb() bool {
	return p.IsMobileWeb() || p.IsDesktopWeb()
}

func (p PlatformType) IsIos() bool {
	return p == PlatformTypeIOS
}

func (p PlatformType) IsAndroid() bool {
	return p == PlatformTypeANDROID
}

func (p PlatformType) IsMobileWeb() bool {
	return p == PlatformTypeWEBMobile || p == PlatformTypeWEBMobile2
}

func (p PlatformType) IsDesktopWeb() bool {
	switch p {
	case PlatformTypeWEB, PlatformTypeDesktop, PlatformTypeWebDesktop, PlatformTypeWebDesktop2:
		return true
	default:
		return false
	}
}
