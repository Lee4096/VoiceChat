package voice

import (
	"math"
	"testing"
)

func TestInt16ToFloat32(t *testing.T) {
	tests := []struct {
		name   string
		input  []int16
		want   float32
		approx float64
	}{
		{"Zero", []int16{0}, 0.0, 0.0},
		{"Max", []int16{32767}, 1.0, 0.0001},
		{"Min", []int16{-32768}, -1.0, 0.0001},
		{"Half", []int16{16383}, 0.5, 0.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Int16ToFloat32(tt.input)
			if len(result) != len(tt.input) {
				t.Errorf("Int16ToFloat32() length = %d, want %d", len(result), len(tt.input))
			}
			got := result[0]
			if math.Abs(float64(got-tt.want)) > tt.approx {
				t.Errorf("Int16ToFloat32()[0] = %v, want ~%v", got, tt.want)
			}
		})
	}
}

func TestFloat32ToInt16(t *testing.T) {
	tests := []struct {
		name  string
		input float32
		want  int16
	}{
		{"Zero", 0.0, 0},
		{"One", 1.0, 32767},
		{"MinusOne", -1.0, -32767},
		{"Half", 0.5, 16383},
		{"ClampAboveOne", 1.5, 32767},
		{"ClampBelowMinusOne", -1.5, -32767},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Float32ToInt16([]float32{tt.input})
			if result[0] != tt.want {
				t.Errorf("Float32ToInt16(%v)[0] = %d, want %d", tt.input, result[0], tt.want)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	original := []int16{-10000, 0, 10000, -32768, 32767}
	float := Int16ToFloat32(original)
	back := Float32ToInt16(float)

	for i, v := range original {
		if math.Abs(float64(back[i]-v)) > 1 {
			t.Errorf("RoundTrip[%d]: %d -> %d, diff > 1", i, v, back[i])
		}
	}
}

func TestBytesToFloat32(t *testing.T) {
	data := []byte{0, 0, 0, 128, 0, 64}
	result := bytesToFloat32(data)

	if len(result) != 3 {
		t.Errorf("bytesToFloat32() length = %d, want 3", len(result))
	}
}

func TestFloat32ToBytes(t *testing.T) {
	samples := []float32{0.0, 1.0, -1.0}
	result := Float32ToBytes(samples)

	if len(result) != 6 {
		t.Errorf("Float32ToBytes() length = %d, want 6", len(result))
	}
}
