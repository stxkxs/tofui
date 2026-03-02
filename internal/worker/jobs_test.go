package worker

import "testing"

func TestPostPlanAction(t *testing.T) {
	tests := []struct {
		name             string
		autoApply        bool
		requiresApproval bool
		want             string
	}{
		{"default: neither set", false, false, "planned"},
		{"auto_apply wins", true, false, "queued"},
		{"requires_approval", false, true, "awaiting_approval"},
		{"both set: auto_apply wins", true, true, "queued"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := postPlanAction(tt.autoApply, tt.requiresApproval)
			if got != tt.want {
				t.Errorf("postPlanAction(%v, %v) = %q, want %q", tt.autoApply, tt.requiresApproval, got, tt.want)
			}
		})
	}
}
