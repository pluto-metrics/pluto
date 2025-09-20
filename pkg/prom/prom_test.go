package prom

import "testing"

func Test_joinPrefix(t *testing.T) {
	type args struct {
		pfx string
		p   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{name: "no prefix",
			args: args{pfx: "/", p: "/api/v1/labels"},
			want: "/api/v1/labels"},
		{name: "with prefix",
			args: args{pfx: "/prom", p: "/api/v1/labels"},
			want: "/prom/api/v1/labels"},
		{name: "with prefix no leading slash",
			args: args{pfx: "prom", p: "/api/v1/labels"},
			want: "/prom/api/v1/labels"},
		{name: "with prefix no trailing slash",
			args: args{pfx: "/prom", p: "api/v1/labels"},
			want: "/prom/api/v1/labels"},
		{name: "with prefix no slashes",
			args: args{pfx: "prom", p: "api/v1/labels"},
			want: "/prom/api/v1/labels"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := joinPrefix(tt.args.pfx, tt.args.p); got != tt.want {
				t.Errorf("joinPrefix(%#v,%#v) = %v, want %#v", tt.args.pfx, tt.args.p, got, tt.want)
			}
		})
	}
}
