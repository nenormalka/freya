package grpc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAddresses(t *testing.T) {
	for name, tt := range map[string]struct {
		addrs   string
		want    map[string][]string
		withErr bool
	}{
		"empty": {
			addrs:   "",
			want:    map[string][]string{},
			withErr: false,
		},
		"valid 1 service 1 address": {
			addrs: "service1=address1:port1",
			want: map[string][]string{
				"service1": {"address1:port1"},
			},
			withErr: false,
		},
		"valid 1 service 2 addresses": {
			addrs: "service1=address1:port1,address2:port2",
			want: map[string][]string{
				"service1": {"address1:port1", "address2:port2"},
			},
			withErr: false,
		},
		"valid 2 services 3 addresses": {
			addrs: "service1=address1:port1,address2:port2;service2=address3:port3",
			want: map[string][]string{
				"service1": {"address1:port1", "address2:port2"},
				"service2": {"address3:port3"},
			},
			withErr: false,
		},
		"invalid 1 service without address #1": {
			addrs:   "service1",
			want:    nil,
			withErr: true,
		},
		"invalid 1 service without address #2": {
			addrs:   "service1=",
			want:    nil,
			withErr: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := getAddresses(tt.addrs)
			if tt.withErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.want, got)
		})
	}
}
