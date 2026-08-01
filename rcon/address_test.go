package rcon

import "testing"

func TestAddressIsUsable(t *testing.T) {
	cases := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:27015", true},
		{"0.0.0.0:27015", true},
		{"121.122.123.124", true},
		{"?.?.?.?:?", false},
		{"", false},
	}

	for _, c := range cases {
		if got := AddressIsValid(c.addr); got != c.want {
			t.Errorf("AddressIsUsable(%q) = %v, want %v", c.addr, got, c.want)
		}
	}
}

func TestGameConnectAddress(t *testing.T) {
	cases := []struct {
		name string
		info ServerInfo
		want string
	}{
		{
			name: "sdr preferred over configured",
			info: ServerInfo{
				ConfiguredAddress: "127.0.0.1:27015",
				Address:           Address{IP: "169.254.12.131:27776", Local: "0.0.0.0:27015"},
			},
			want: "169.254.12.131:27776",
		},
		{
			name: "usable local is preferred over configured",
			info: ServerInfo{
				ConfiguredAddress: "127.0.0.1:27015",
				Address:           Address{IP: "?.?.?.?:?", Local: "0.0.0.0:27015"},
			},
			want: "0.0.0.0:27015",
		},
		{
			name: "usable local falls back",
			info: ServerInfo{
				Address: Address{IP: "?.?.?.?:?", Local: "192.168.1.5:27015"},
			},
			want: "192.168.1.5:27015",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.info.GameConnectAddress(); got != c.want {
				t.Errorf("GameConnectAddress() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestSTVConnectAddress(t *testing.T) {
	cases := []struct {
		name string
		info ServerInfo
		want string
	}{
		{
			name: "stv address preferred",
			info: ServerInfo{
				ConfiguredAddress: "127.0.0.1:27015",
				SourceTV:          SourceTV{Address: "169.254.12.131:27776", Delay: "30.0s", Local: "0.0.0.0:27020"},
			},
			want: "169.254.12.131:27776",
		},
		{
			name: "stv local fallback keeps zero bind",
			info: ServerInfo{
				ConfiguredAddress: "127.0.0.1:27015",
				SourceTV:          SourceTV{Address: "?.?.?.?:?", Delay: "30.0s", Local: "0.0.0.0:27020"},
			},
			want: "0.0.0.0:27020",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.info.STVConnectAddress(); got != c.want {
				t.Errorf("STVConnectAddress() = %q, want %q", got, c.want)
			}
		})
	}
}
