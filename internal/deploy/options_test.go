package deploy

import "testing"

func TestOptions(t *testing.T) {
	tests := []struct {
		name       string
		opts       *Options
		wantConfig string
		wantNoWait bool
		wantTime   int
	}{
		{
			name: "default options",
			opts: &Options{
				ConfigFile: "apcdeploy.yml",
				NoWait:     false,
				Timeout:    300,
			},
			wantConfig: "apcdeploy.yml",
			wantNoWait: false,
			wantTime:   300,
		},
		{
			name: "custom options",
			opts: &Options{
				ConfigFile: "custom.yml",
				NoWait:     true,
				Timeout:    600,
			},
			wantConfig: "custom.yml",
			wantNoWait: true,
			wantTime:   600,
		},
		{
			name: "zero timeout",
			opts: &Options{
				ConfigFile: "apcdeploy.yml",
				NoWait:     false,
				Timeout:    0,
			},
			wantConfig: "apcdeploy.yml",
			wantNoWait: false,
			wantTime:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.opts.ConfigFile != tt.wantConfig {
				t.Errorf("ConfigFile = %v, want %v", tt.opts.ConfigFile, tt.wantConfig)
			}
			if tt.opts.NoWait != tt.wantNoWait {
				t.Errorf("NoWait = %v, want %v", tt.opts.NoWait, tt.wantNoWait)
			}
			if tt.opts.Timeout != tt.wantTime {
				t.Errorf("Timeout = %v, want %v", tt.opts.Timeout, tt.wantTime)
			}
		})
	}
}
