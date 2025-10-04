package deploy

import "testing"

func TestOptions(t *testing.T) {
	tests := []struct {
		name       string
		opts       *Options
		wantConfig string
		wantWait   bool
		wantTime   int
	}{
		{
			name: "default options",
			opts: &Options{
				ConfigFile: "apcdeploy.yml",
				Wait:       false,
				Timeout:    600,
			},
			wantConfig: "apcdeploy.yml",
			wantWait:   false,
			wantTime:   600,
		},
		{
			name: "custom options",
			opts: &Options{
				ConfigFile: "custom.yml",
				Wait:       true,
				Timeout:    600,
			},
			wantConfig: "custom.yml",
			wantWait:   true,
			wantTime:   600,
		},
		{
			name: "zero timeout",
			opts: &Options{
				ConfigFile: "apcdeploy.yml",
				Wait:       false,
				Timeout:    0,
			},
			wantConfig: "apcdeploy.yml",
			wantWait:   false,
			wantTime:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.opts.ConfigFile != tt.wantConfig {
				t.Errorf("ConfigFile = %v, want %v", tt.opts.ConfigFile, tt.wantConfig)
			}
			if tt.opts.Wait != tt.wantWait {
				t.Errorf("Wait = %v, want %v", tt.opts.Wait, tt.wantWait)
			}
			if tt.opts.Timeout != tt.wantTime {
				t.Errorf("Timeout = %v, want %v", tt.opts.Timeout, tt.wantTime)
			}
		})
	}
}
