package ws

import (
	"testing"
)

func TestTryParseBestBidAskObject(t *testing.T) {
	t.Parallel()

	full := BestBidAskEvent{
		Market:    "m1",
		AssetID:   "tok1",
		BestBid:   "0.5",
		BestAsk:   "0.6",
		Spread:    "0.1",
		Timestamp: "ts",
	}

	tests := []struct {
		name   string
		json   string
		wantOK bool
		want   BestBidAskEvent
	}{
		{
			name:   "event_type_all_fields",
			json:   `{"event_type":"best_bid_ask","asset_id":"tok1","market":"m1","best_bid":"0.5","best_ask":"0.6","spread":"0.1","timestamp":"ts"}`,
			wantOK: true,
			want:   full,
		},
		{
			name:   "type_alias_instead_of_event_type",
			json:   `{"type":"best_bid_ask","asset_id":"tok1","best_bid":"0.5","best_ask":"0.6"}`,
			wantOK: true,
			want: BestBidAskEvent{
				AssetID: "tok1",
				BestBid: "0.5",
				BestAsk: "0.6",
			},
		},
		{
			name:   "type_is_event_kind_timestamp_separate",
			json:   `{"type":"best_bid_ask","asset_id":"a","timestamp":"2024-01-01"}`,
			wantOK: true,
			want: BestBidAskEvent{
				AssetID:   "a",
				Timestamp: "2024-01-01",
			},
		},
		{
			name:   "event_type_after_other_fields",
			json:   `{"asset_id":"tok1","best_bid":"1","best_ask":"2","event_type":"best_bid_ask"}`,
			wantOK: true,
			want: BestBidAskEvent{
				AssetID: "tok1",
				BestBid: "1",
				BestAsk: "2",
			},
		},
		{
			name:   "whitespace_after_colon",
			json:   "{\"event_type\":  \t \"best_bid_ask\" , \"asset_id\" : \"x\"}",
			wantOK: true,
			want:   BestBidAskEvent{AssetID: "x"},
		},
		{
			name:   "wrong_event_type",
			json:   `{"event_type":"book","asset_id":"tok1"}`,
			wantOK: false,
		},
		{
			name:   "missing_event_type",
			json:   `{"asset_id":"tok1","best_bid":"0.5"}`,
			wantOK: false,
		},
		{
			name:   "empty_object",
			json:   `{}`,
			wantOK: false,
		},
		{
			name:   "non_string_asset_id_skipped",
			json:   `{"event_type":"best_bid_ask","asset_id":12345}`,
			wantOK: true,
			want:   BestBidAskEvent{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got BestBidAskEvent
			ok := tryParseBestBidAskObject([]byte(tt.json), &got)
			if ok != tt.wantOK {
				t.Fatalf("tryParseBestBidAskObject() ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if got != tt.want {
				t.Fatalf("parsed event:\n got  %+v\n want %+v", got, tt.want)
			}
		})
	}
}

func TestTryParseBestBidAskObject_event_type_wins_over_type(t *testing.T) {
	t.Parallel()
	// Later key wins: type=book after event_type=best_bid_ask → not a BBO payload.
	json := `{"event_type":"best_bid_ask","type":"book","asset_id":"z"}`
	var got BestBidAskEvent
	if tryParseBestBidAskObject([]byte(json), &got) {
		t.Fatal("expected ok false when type overwrites event_type")
	}
}

func TestTryParseBestBidAskObject_type_then_event_type_best_bid_ask(t *testing.T) {
	t.Parallel()
	json := `{"type":"book","event_type":"best_bid_ask","asset_id":"z"}`
	var got BestBidAskEvent
	if !tryParseBestBidAskObject([]byte(json), &got) {
		t.Fatal("expected ok true")
	}
	if got.AssetID != "z" {
		t.Fatalf("AssetID = %q", got.AssetID)
	}
}
