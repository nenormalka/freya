package metadata

import (
	"context"
	"errors"
	"fmt"
	"math"

	v "github.com/hashicorp/go-version"
	"github.com/mailru/easyjson"
	_ "github.com/mailru/easyjson/gen"

	"google.golang.org/grpc/metadata"
)

type (
	PlatformType string
	Ratio        int

	MetadataError    error
	VersionDataError error

	VersionData map[PlatformType]string
)

//go:generate easyjson

//easyjson:json
type (
	Pair struct {
		Key   string
		Value string
	}

	Pairs []Pair
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

func GetDatesFromCtx(ctx context.Context, key string) ([]string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, ErrEmptyMetadata
	}

	dates, ok := md[key]
	if !ok {
		return nil, ErrKeyNotFound
	}

	if len(dates) == 0 {
		return nil, ErrEmptyValuesInKey
	}

	return dates, nil
}

func GetDataFromCtx(ctx context.Context, key string) (string, error) {
	dates, err := GetDatesFromCtx(ctx, key)
	if err != nil {
		return "", fmt.Errorf("get dates from context error: %w", err)
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

func AddKeyValueToCtx(ctx context.Context, metaKey, key, value string) (context.Context, error) {
	data, err := easyjson.Marshal(Pair{Key: key, Value: value})
	if err != nil {
		return ctx, fmt.Errorf("marshal key value error: %w", err)
	}

	return metadata.AppendToOutgoingContext(ctx, metaKey, string(data)), nil
}

func AddPairsToCtx(ctx context.Context, metaKey string, pairs Pairs) (context.Context, error) {
	for _, pair := range pairs {
		var err error
		ctx, err = AddKeyValueToCtx(ctx, metaKey, pair.Key, pair.Value)
		if err != nil {
			return ctx, fmt.Errorf("add key value to context error: %w", err)
		}
	}

	return ctx, nil
}

func GetPairsFromCtx(ctx context.Context, metaKey string) (Pairs, error) {
	dates, err := GetDatesFromCtx(ctx, metaKey)
	if err != nil {
		return nil, fmt.Errorf("get data from context error: %w", err)
	}

	pairs := make([]Pair, 0, len(dates))
	for _, date := range dates {
		pair := Pair{}
		if err = easyjson.Unmarshal([]byte(date), &pair); err != nil {
			return nil, fmt.Errorf("unmarshal key value error: %w", err)
		}

		pairs = append(pairs, pair)
	}

	return pairs, nil
}

func GetValueFromCtx(ctx context.Context, metaKey, key string) (string, error) {
	pairs, err := GetPairsFromCtx(ctx, metaKey)
	if err != nil {
		return "", fmt.Errorf("get pairs from context error: %w", err)
	}

	for _, pair := range pairs {
		if pair.Key == key {
			return pair.Value, nil
		}
	}

	return "", ErrKeyNotFound
}

func GetValueFromCtxWithDefault(ctx context.Context, metaKey, key, defaultValue string) (string, error) {
	value, err := GetValueFromCtx(ctx, metaKey, key)
	if err != nil {
		return defaultValue, nil
	}

	return value, nil
}

func GetPairsFromCtxWithKey(ctx context.Context, metaKey, key string) (Pairs, error) {
	pairs, err := GetPairsFromCtx(ctx, metaKey)
	if err != nil {
		return nil, fmt.Errorf("get pairs from context error: %w", err)
	}

	result := make([]Pair, 0, len(pairs))
	for _, pair := range pairs {
		if pair.Key == key {
			result = append(result, pair)
		}
	}

	return result, nil
}
