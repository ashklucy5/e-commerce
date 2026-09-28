package supportattachment

import (
	"testing"
	"time"
)

func TestNormalizeRetentionBatchLimit(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name string

		input int

		want int
	}{
		{
			name: "default zero",

			input: 0,

			want: DefaultRetentionBatchSize,
		},
		{
			name: "default negative",

			input: -1,

			want: DefaultRetentionBatchSize,
		},
		{
			name: "normal",

			input: 50,

			want: 50,
		},
		{
			name: "maximum",

			input: MaxRetentionBatchSize,

			want: MaxRetentionBatchSize,
		},
		{
			name: "clamped maximum",

			input: MaxRetentionBatchSize + 1,

			want: MaxRetentionBatchSize,
		},
	}

	for _, test := range tests {

		test := test

		t.Run(
			test.name,
			func(t *testing.T) {
				t.Parallel()

				got :=
					normalizeRetentionBatchLimit(
						test.input,
					)

				if got !=
					test.want {

					t.Fatalf(
						"limit = %d, want %d",
						got,
						test.want,
					)
				}
			},
		)
	}
}

func TestRetentionCutoffs(
	t *testing.T,
) {
	t.Parallel()

	now :=
		time.Date(
			2026,
			time.September,
			27,
			12,
			30,
			0,
			0,
			time.FixedZone(
				"test",
				8*60*60,
			),
		)

	closedBefore,
		orphanBefore :=
		retentionCutoffs(
			now,
		)

	wantNow :=
		now.UTC()

	wantClosed :=
		wantNow.Add(
			-7 * 24 * time.Hour,
		)

	wantOrphan :=
		wantNow.Add(
			-24 * time.Hour,
		)

	if !closedBefore.Equal(
		wantClosed,
	) {
		t.Fatalf(
			"closed cutoff = %s, want %s",
			closedBefore,
			wantClosed,
		)
	}

	if !orphanBefore.Equal(
		wantOrphan,
	) {
		t.Fatalf(
			"orphan cutoff = %s, want %s",
			orphanBefore,
			wantOrphan,
		)
	}
}

func TestRetentionDurations(
	t *testing.T,
) {
	t.Parallel()

	if RetentionAfterCaseClose !=
		7*24*time.Hour {

		t.Fatalf(
			"case retention = %s, want 7 days",
			RetentionAfterCaseClose,
		)
	}

	if OrphanRetentionAge !=
		24*time.Hour {

		t.Fatalf(
			"orphan retention = %s, want 24 hours",
			OrphanRetentionAge,
		)
	}
}
