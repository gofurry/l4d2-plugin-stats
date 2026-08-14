package service

import "testing"

func TestIsFirearmEquipment(t *testing.T) {
	t.Parallel()
	tests := []struct {
		id   int64
		want bool
	}{
		{1, true}, {20, true}, {21, false}, {22, true}, {23, true}, {24, false}, {39, false},
	}
	for _, test := range tests {
		if got := isFirearmEquipment(test.id); got != test.want {
			t.Errorf("isFirearmEquipment(%d) = %t, want %t", test.id, got, test.want)
		}
	}
}
