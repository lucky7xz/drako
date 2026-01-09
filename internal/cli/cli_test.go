package cli

import (
	"reflect"
	"testing"
)

func TestParsePurgeFlags(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedOpts  PurgeOptions
		expectedInter bool
		expectError   bool
	}{
		{
			name: "Target Config Full",
			args: []string{"--config"},
			expectedOpts: PurgeOptions{
				TargetConfig: true,
			},
			expectedInter: false,
			expectError:   false,
		},
		{
			name: "Target Config Alias",
			args: []string{"-c"},
			expectedOpts: PurgeOptions{
				TargetConfig: true,
			},
			expectedInter: false,
			expectError:   false,
		},
		{
			name: "Target Core Profile",
			args: []string{"--target", "core"},
			expectedOpts: PurgeOptions{
				TargetProfile: "core",
			},
			expectedInter: false,
			expectError:   false,
		},
		{
			name: "Target Profile Alias",
			args: []string{"-t", "git"},
			expectedOpts: PurgeOptions{
				TargetProfile: "git",
			},
			expectedInter: false,
			expectError:   false,
		},
		{
			name:         "Interactive Mode",
			args:         []string{"--interactive"},
			expectedOpts: PurgeOptions{
				// No target set initially in interactive
			},
			expectedInter: true,
			expectError:   false,
		},
		{
			name: "Destroy Everything",
			args: []string{"--destroyeverything"},
			expectedOpts: PurgeOptions{
				DestroyEverything: true,
			},
			expectedInter: false,
			expectError:   false,
		},
		{
			name: "Positional Args Error",
			args: []string{"some_file"},
			// Should error out
			expectError: true,
		},
		{
			name: "Positional Arg 'ssh' (User Report)",
			args: []string{"ssh"},
			// Should error out explicitly
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, interactive, err := ParsePurgeFlags(tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(*opts, tt.expectedOpts) {
				t.Errorf("Options mismatch.\nGot: %+v\nWant: %+v", *opts, tt.expectedOpts)
			}

			if interactive != tt.expectedInter {
				t.Errorf("Interactive mismatch. Got %v, Want %v", interactive, tt.expectedInter)
			}
		})
	}
}
