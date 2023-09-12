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
