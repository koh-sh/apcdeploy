package run

import "testing"

func TestOptions(t *testing.T) {
	tests := []struct {
		name           string
		opts           *Options
		wantConfig     string
		wantWaitDeploy bool
		wantWaitBake   bool
		wantTime       int
	}{
		{
			name: "default options",
			opts: &Options{
				ConfigFile: "apcdeploy.yml",
				WaitDeploy: false,
				WaitBake:   false,
				Timeout:    600,
			},
			wantConfig:     "apcdeploy.yml",
			wantWaitDeploy: false,
			wantWaitBake:   false,
			wantTime:       600,
		},
		{
			name: "wait for deploy only",
			opts: &Options{
				ConfigFile: "custom.yml",
				WaitDeploy: true,
				WaitBake:   false,
				Timeout:    600,
			},
			wantConfig:     "custom.yml",
			wantWaitDeploy: true,
			wantWaitBake:   false,
			wantTime:       600,
		},
		{
			name: "wait for bake (complete)",
			opts: &Options{
				ConfigFile: "custom.yml",
				WaitDeploy: false,
				WaitBake:   true,
				Timeout:    600,
			},
			wantConfig:     "custom.yml",
			wantWaitDeploy: false,
			wantWaitBake:   true,
			wantTime:       600,
		},
		{
			name: "zero timeout",
			opts: &Options{
				ConfigFile: "apcdeploy.yml",
				WaitDeploy: false,
				WaitBake:   false,
				Timeout:    0,
			},
			wantConfig:     "apcdeploy.yml",
			wantWaitDeploy: false,
			wantWaitBake:   false,
			wantTime:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.opts.ConfigFile != tt.wantConfig {
				t.Errorf("ConfigFile = %v, want %v", tt.opts.ConfigFile, tt.wantConfig)
			}
			if tt.opts.WaitDeploy != tt.wantWaitDeploy {
				t.Errorf("WaitDeploy = %v, want %v", tt.opts.WaitDeploy, tt.wantWaitDeploy)
			}
			if tt.opts.WaitBake != tt.wantWaitBake {
				t.Errorf("WaitBake = %v, want %v", tt.opts.WaitBake, tt.wantWaitBake)
			}
			if tt.opts.Timeout != tt.wantTime {
				t.Errorf("Timeout = %v, want %v", tt.opts.Timeout, tt.wantTime)
			}
		})
	}
}
