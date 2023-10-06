package metadata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestCompareVersions(t *testing.T) {
	for name, tt := range map[string]struct {
		ctx     context.Context
		vd      VersionData
		want    Ratio
		wantErr bool
		err     error
	}{
		"equal ios": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					AppInfoFieldPlatform:   []string{string(PlatformTypeIOS)},
					AppInfoFieldAppVersion: []string{"14"},
				})
			}(),
			vd: VersionData{
				PlatformTypeIOS:     "14.0.0",
				PlatformTypeANDROID: "9.5.12",
				PlatformTypeWEB:     "0.2.7",
			},
			want:    RatioEqual,
			wantErr: false,
			err:     nil,
		},
		"great ios": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					AppInfoFieldPlatform:   []string{string(PlatformTypeIOS)},
					AppInfoFieldAppVersion: []string{"15"},
				})
			}(),
			vd: VersionData{
				PlatformTypeIOS:     "14.0.0",
				PlatformTypeANDROID: "9.5.12",
				PlatformTypeWEB:     "0.2.7",
			},
			want:    RatioGreat,
			wantErr: false,
			err:     nil,
		},
		"less ios": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					AppInfoFieldPlatform:   []string{string(PlatformTypeIOS)},
					AppInfoFieldAppVersion: []string{"14"},
				})
			}(),
			vd: VersionData{
				PlatformTypeIOS:     "14.0.1",
				PlatformTypeANDROID: "9.5.12",
				PlatformTypeWEB:     "0.2.7",
			},
			want:    RatioLess,
			wantErr: false,
			err:     nil,
		},
		"equal android": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					AppInfoFieldPlatform:   []string{string(PlatformTypeANDROID)},
					AppInfoFieldAppVersion: []string{"9.5.12"},
				})
			}(),
			vd: VersionData{
				PlatformTypeIOS:     "14.0.0",
				PlatformTypeANDROID: "9.5.12",
				PlatformTypeWEB:     "0.2.7",
			},
			want:    RatioEqual,
			wantErr: false,
			err:     nil,
		},
		"great android": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					AppInfoFieldPlatform:   []string{string(PlatformTypeANDROID)},
					AppInfoFieldAppVersion: []string{"9.6.1"},
				})
			}(),
			vd: VersionData{
				PlatformTypeIOS:     "14.0.0",
				PlatformTypeANDROID: "9.5.12",
				PlatformTypeWEB:     "0.2.7",
			},
			want:    RatioGreat,
			wantErr: false,
			err:     nil,
		},
		"less android": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					AppInfoFieldPlatform:   []string{string(PlatformTypeANDROID)},
					AppInfoFieldAppVersion: []string{"9.5.11"},
				})
			}(),
			vd: VersionData{
				PlatformTypeIOS:     "14.0.1",
				PlatformTypeANDROID: "9.5.12",
				PlatformTypeWEB:     "0.2.7",
			},
			want:    RatioLess,
			wantErr: false,
			err:     nil,
		},
		"equal web": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					AppInfoFieldPlatform:   []string{string(PlatformTypeWEB)},
					AppInfoFieldAppVersion: []string{"0.2.7"},
				})
			}(),
			vd: VersionData{
				PlatformTypeIOS:     "14.0.0",
				PlatformTypeANDROID: "9.5.12",
				PlatformTypeWEB:     "0.2.7",
			},
			want:    RatioEqual,
			wantErr: false,
			err:     nil,
		},
		"great web": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					AppInfoFieldPlatform:   []string{string(PlatformTypeWEB)},
					AppInfoFieldAppVersion: []string{"1"},
				})
			}(),
			vd: VersionData{
				PlatformTypeIOS:     "14.0.0",
				PlatformTypeANDROID: "9.5.12",
				PlatformTypeWEB:     "0.2.7",
			},
			want:    RatioGreat,
			wantErr: false,
			err:     nil,
		},
		"less web": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					AppInfoFieldPlatform:   []string{string(PlatformTypeWEB)},
					AppInfoFieldAppVersion: []string{"0.1"},
				})
			}(),
			vd: VersionData{
				PlatformTypeIOS:     "14.0.1",
				PlatformTypeANDROID: "9.5.12",
				PlatformTypeWEB:     "0.2.7",
			},
			want:    RatioLess,
			wantErr: false,
			err:     nil,
		},
		"empty metadata": {
			ctx: context.Background(),
			vd: VersionData{
				PlatformTypeIOS:     "14.0.1",
				PlatformTypeANDROID: "9.5.12",
				PlatformTypeWEB:     "0.2.7",
			},
			want:    RatioError,
			wantErr: true,
			err:     ErrEmptyMetadata,
		},
		"empty platform md": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					AppInfoFieldAppVersion: []string{"0.1"},
				})
			}(),
			vd: VersionData{
				PlatformTypeIOS:     "14.0.1",
				PlatformTypeANDROID: "9.5.12",
				PlatformTypeWEB:     "0.2.7",
			},
			want:    RatioError,
			wantErr: true,
			err:     ErrKeyNotFound,
		},
		"empty version md": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					AppInfoFieldPlatform: []string{string(PlatformTypeWEB)},
				})
			}(),
			vd: VersionData{
				PlatformTypeIOS:     "14.0.1",
				PlatformTypeANDROID: "9.5.12",
				PlatformTypeWEB:     "0.2.7",
			},
			want:    RatioError,
			wantErr: true,
			err:     ErrKeyNotFound,
		},
		"empty platform vd": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					AppInfoFieldPlatform:   []string{string(PlatformTypeWEB)},
					AppInfoFieldAppVersion: []string{"0.1"},
				})
			}(),
			vd:      nil,
			want:    RatioError,
			wantErr: true,
			err:     ErrEmptyCondition,
		},
		"invalid version in condition": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					AppInfoFieldPlatform:   []string{string(PlatformTypeWEB)},
					AppInfoFieldAppVersion: []string{"0.1"},
				})
			}(),
			vd: VersionData{
				PlatformTypeIOS:     "14.0.1",
				PlatformTypeANDROID: "9.5.12",
				PlatformTypeWEB:     "",
			},
			want:    RatioError,
			wantErr: true,
			err:     ErrConditionVersion,
		},
		"invalid version in context": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					AppInfoFieldPlatform:   []string{string(PlatformTypeWEB)},
					AppInfoFieldAppVersion: []string{""},
				})
			}(),
			vd: VersionData{
				PlatformTypeIOS:     "14.0.1",
				PlatformTypeANDROID: "9.5.12",
				PlatformTypeWEB:     "0.1",
			},
			want:    RatioError,
			wantErr: true,
			err:     ErrContextVersion,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := CompareVersions(tt.ctx, tt.vd)
			if tt.wantErr {
				require.NotNil(t, err)

				if tt.err != nil {
					require.Equal(t, tt.err, err)
				}
			} else {
				require.Nil(t, err)
			}

			require.Equal(t, tt.want, got)
		})
	}
}

func TestFeatureToggleIsEnabled(t *testing.T) {
	for name, tt := range map[string]struct {
		ctx    context.Context
		toggle int
		want   bool
	}{
		"empty metadata": {
			ctx:    context.Background(),
			toggle: 1,
			want:   false,
		},
		"wrong toggle": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					"feature-toggle-1": []string{"enabled"},
				})
			}(),
			toggle: 2,
			want:   false,
		},
		"toggle disabled": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					"feature-toggle-1": []string{"disabled"},
				})
			}(),
			toggle: 1,
			want:   false,
		},
		"toggle enabled": {
			ctx: func() context.Context {
				ctx := context.Background()
				return metadata.NewIncomingContext(ctx, metadata.MD{
					"feature-toggle-1": []string{"enabled"},
				})
			}(),
			toggle: 1,
			want:   true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := FeatureToggleIsEnabled(tt.ctx, tt.toggle); got != tt.want {
				t.Errorf("FeatureToggleIsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPlatformType(t *testing.T) {
	setPlatform := func(platform string) context.Context {
		return metadata.NewIncomingContext(context.Background(), metadata.MD{
			AppInfoFieldPlatform: []string{platform},
		})
	}

	for name, tt := range map[string]struct {
		platform        string
		wantIsValid     bool
		wantIsIos       bool
		wantIsAndroid   bool
		wantIsWeb       bool
		wantIsDesktop   bool
		wantIsMobile    bool
		wantIsMobileWeb bool
	}{
		"empty metadata": {
			platform:        "",
			wantIsValid:     false,
			wantIsIos:       false,
			wantIsAndroid:   false,
			wantIsWeb:       false,
			wantIsDesktop:   false,
			wantIsMobile:    false,
			wantIsMobileWeb: false,
		},
		"ios": {
			platform:        "ios",
			wantIsValid:     true,
			wantIsIos:       true,
			wantIsAndroid:   false,
			wantIsWeb:       false,
			wantIsDesktop:   false,
			wantIsMobile:    true,
			wantIsMobileWeb: false,
		},
		"android": {
			platform:        "android",
			wantIsValid:     true,
			wantIsIos:       false,
			wantIsAndroid:   true,
			wantIsWeb:       false,
			wantIsDesktop:   false,
			wantIsMobile:    true,
			wantIsMobileWeb: false,
		},
		"web": {
			platform:        "web",
			wantIsValid:     true,
			wantIsIos:       false,
			wantIsAndroid:   false,
			wantIsWeb:       true,
			wantIsDesktop:   true,
			wantIsMobile:    false,
			wantIsMobileWeb: false,
		},
		"web-mobile": {
			platform:        "web-mobile",
			wantIsValid:     true,
			wantIsIos:       false,
			wantIsAndroid:   false,
			wantIsWeb:       true,
			wantIsDesktop:   false,
			wantIsMobile:    false,
			wantIsMobileWeb: true,
		},
		"web_mobile": {
			platform:        "web-mobile",
			wantIsValid:     true,
			wantIsIos:       false,
			wantIsAndroid:   false,
			wantIsWeb:       true,
			wantIsDesktop:   false,
			wantIsMobile:    false,
			wantIsMobileWeb: true,
		},
		"desktop": {
			platform:        "desktop",
			wantIsValid:     true,
			wantIsIos:       false,
			wantIsAndroid:   false,
			wantIsWeb:       true,
			wantIsDesktop:   true,
			wantIsMobile:    false,
			wantIsMobileWeb: false,
		},
		"web-desktop": {
			platform:        "web-desktop",
			wantIsValid:     true,
			wantIsIos:       false,
			wantIsAndroid:   false,
			wantIsWeb:       true,
			wantIsDesktop:   true,
			wantIsMobile:    false,
			wantIsMobileWeb: false,
		},
		"web_desktop": {
			platform:        "web_desktop",
			wantIsValid:     true,
			wantIsIos:       false,
			wantIsAndroid:   false,
			wantIsWeb:       true,
			wantIsDesktop:   true,
			wantIsMobile:    false,
			wantIsMobileWeb: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := GetPlatformType(setPlatform(tt.platform))
			require.Nil(t, err)

			if got.IsValid() != tt.wantIsValid {
				t.Errorf("IsValid() = %v, want %v", got.IsValid(), tt.wantIsValid)
			}

			if got.IsIos() != tt.wantIsIos {
				t.Errorf("IsIos() = %v, want %v", got.IsIos(), tt.wantIsIos)
			}

			if got.IsAndroid() != tt.wantIsAndroid {
				t.Errorf("IsAndroid() = %v, want %v", got.IsAndroid(), tt.wantIsAndroid)
			}

			if got.IsMobile() != tt.wantIsMobile {
				t.Errorf("IsMobile() = %v, want %v", got.IsMobile(), tt.wantIsMobile)
			}

			if got.IsMobileWeb() != tt.wantIsMobileWeb {
				t.Errorf("IsMobileWeb() = %v, want %v", got.IsMobileWeb(), tt.wantIsMobileWeb)
			}

			if got.IsDesktopWeb() != tt.wantIsDesktop {
				t.Errorf("IsDesktopWeb() = %v, want %v", got.IsDesktopWeb(), tt.wantIsDesktop)
			}

			if got.IsWeb() != tt.wantIsWeb {
				t.Errorf("IsWeb() = %v, want %v", got.IsWeb(), tt.wantIsWeb)
			}
		})
	}
}
