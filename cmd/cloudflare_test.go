package cmd

import (
	"net"
	"testing"

	"github.com/spf13/cobra"
)

func Test_flareValidate(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := flareValidate(); (err != nil) != tt.wantErr {
				t.Errorf("flareValidate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_updateDNSARecords(t *testing.T) {
	type args struct {
		cIP net.IP
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := updateDNSARecords(tt.args.cIP); (err != nil) != tt.wantErr {
				t.Errorf("updateDNSARecords() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_flareRun(t *testing.T) {
	type args struct {
		cmd  *cobra.Command
		args []string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flareRun(tt.args.cmd, tt.args.args)
		})
	}
}

func Test_matchFQDNFilter(t *testing.T) {
	type args struct {
		filters  []string
		fqdn     string
		emptyval bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "emptyval:true no filters",
			args: args{
				filters:  []string{},
				emptyval: true,
			},
			want: true,
		},
		{
			name: "emptyval:false no filters",
			args: args{
				filters:  []string{},
				emptyval: false,
			},
			want: false,
		},
		{
			name: "one filters success filter",
			args: args{
				filters: []string{"sample"},
				fqdn:    "sample.com",
			},
			want: true,
		},
		{
			name: "one filters fail filter",
			args: args{
				filters: []string{"sample"},
				fqdn:    "apple.jp",
			},
			want: false,
		},
		{
			name: "two filters success filter",
			args: args{
				filters: []string{"sample", "apple"},
				fqdn:    "sample.com",
			},
			want: true,
		},
		{
			name: "two filters fail filter",
			args: args{
				filters: []string{"sample", "banana"},
				fqdn:    "apple.jp",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchFQDNFilter(tt.args.filters, tt.args.fqdn, tt.args.emptyval); got != tt.want {
				t.Errorf("%s:matchFQDNFilter() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
