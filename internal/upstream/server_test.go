package upstream

import (
	"errors"
	"reflect"
	"testing"
)

func TestSplitCommandLine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		command   string
		want      []string
		expectErr bool
	}{
		{
			name:    "simple command",
			command: "/bin/echo",
			want:    []string{"/bin/echo"},
		},
		{
			name:    "command with arguments",
			command: "/usr/bin/env bash -c \"echo hello world\"",
			want:    []string{"/usr/bin/env", "bash", "-c", "echo hello world"},
		},
		{
			name:    "handles escaped space",
			command: "/bin/echo some\\ value",
			want:    []string{"/bin/echo", "some value"},
		},
		{
			name:      "unterminated quote",
			command:   "/bin/echo \"unterminated",
			expectErr: true,
		},
		{
			name:      "unterminated escape",
			command:   "/bin/echo trailing\\",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := splitCommandLine(tc.command)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("unexpected result\nwant: %#v\ngot:  %#v", tc.want, got)
			}
		})
	}
}

func TestNormalizeCommand(t *testing.T) {
	t.Parallel()

	rbenvCommand := "/home/ec2-user/.rbenv/bin/rbenv exec bundle exec --keep-file-descriptors puma -C /var/www/oauth2id/shared/puma.rb"
	base, args, err := normalizeCommand(rbenvCommand, nil)
	if err != nil {
		t.Fatalf("normalizeCommand returned error: %v", err)
	}

	expectedBase := "/home/ec2-user/.rbenv/bin/rbenv"
	expectedArgs := []string{"exec", "bundle", "exec", "--keep-file-descriptors", "puma", "-C", "/var/www/oauth2id/shared/puma.rb"}

	if base != expectedBase {
		t.Fatalf("unexpected base command, want %q got %q", expectedBase, base)
	}

	if !reflect.DeepEqual(args, expectedArgs) {
		t.Fatalf("unexpected arguments\nwant: %#v\ngot:  %#v", expectedArgs, args)
	}

	_, _, err = normalizeCommand("  ", nil)
	if !errors.Is(err, errEmptyCommand) {
		t.Fatalf("expected upstream command is empty error, got %v", err)
	}
}
