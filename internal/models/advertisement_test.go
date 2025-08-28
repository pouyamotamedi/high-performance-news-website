package models

import (
	"testing"
	"time"
)

func TestAdvertisementCampaign_IsValidCampaign(t *testing.T) {
	tests := []struct {
		name     string
		campaign AdvertisementCampaign
		wantErr  bool
	}{
		{
			name: "valid campaign",
			campaign: AdvertisementCampaign{
				Name:           "Test Campaign",
				AdvertiserName: "Test Advertiser",
				StartDate:      time.Now(),
				Priority:       5,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			campaign: AdvertisementCampaign{
				AdvertiserName: "Test Advertiser",
				StartDate:      time.Now(),
				Priority:       5,
			},
			wantErr: true,
		},
		{
			name: "missing advertiser name",
			campaign: AdvertisementCampaign{
				Name:      "Test Campaign",
				StartDate: time.Now(),
				Priority:  5,
			},
			wantErr: true,
		},
		{
			name: "missing start date",
			campaign: AdvertisementCampaign{
				Name:           "Test Campaign",
				AdvertiserName: "Test Advertiser",
				Priority:       5,
			},
			wantErr: true,
		},
		{
			name: "invalid priority - too low",
			campaign: AdvertisementCampaign{
				Name:           "Test Campaign",
				AdvertiserName: "Test Advertiser",
				StartDate:      time.Now(),
				Priority:       0,
			},
			wantErr: true,
		},
		{
			name: "invalid priority - too high",
			campaign: AdvertisementCampaign{
				Name:           "Test Campaign",
				AdvertiserName: "Test Advertiser",
				StartDate:      time.Now(),
				Priority:       11,
			},
			wantErr: true,
		},
		{
			name: "end date before start date",
			campaign: AdvertisementCampaign{
				Name:           "Test Campaign",
				AdvertiserName: "Test Advertiser",
				StartDate:      time.Now(),
				EndDate:        &[]time.Time{time.Now().AddDate(0, 0, -1)}[0],
				Priority:       5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.campaign.IsValidCampaign()
			if (err != nil) != tt.wantErr {
				t.Errorf("AdvertisementCampaign.IsValidCampaign() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAdvertisementSlot_IsValidSlot(t *testing.T) {
	tests := []struct {
		name    string
		slot    AdvertisementSlot
		wantErr bool
	}{
		{
			name: "valid slot",
			slot: AdvertisementSlot{
				Name:     "Test Slot",
				Slug:     "test-slot",
				PageType: "homepage",
				Position: "header",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			slot: AdvertisementSlot{
				Slug:     "test-slot",
				PageType: "homepage",
				Position: "header",
			},
			wantErr: true,
		},
		{
			name: "missing slug",
			slot: AdvertisementSlot{
				Name:     "Test Slot",
				PageType: "homepage",
				Position: "header",
			},
			wantErr: true,
		},
		{
			name: "invalid page type",
			slot: AdvertisementSlot{
				Name:     "Test Slot",
				Slug:     "test-slot",
				PageType: "invalid",
				Position: "header",
			},
			wantErr: true,
		},
		{
			name: "invalid position",
			slot: AdvertisementSlot{
				Name:     "Test Slot",
				Slug:     "test-slot",
				PageType: "homepage",
				Position: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.slot.IsValidSlot()
			if (err != nil) != tt.wantErr {
				t.Errorf("AdvertisementSlot.IsValidSlot() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAdvertisementCreative_IsValidCreative(t *testing.T) {
	tests := []struct {
		name     string
		creative AdvertisementCreative
		wantErr  bool
	}{
		{
			name: "valid image creative",
			creative: AdvertisementCreative{
				Name:    "Test Creative",
				Type:    "image",
				Content: "https://example.com/image.jpg",
			},
			wantErr: false,
		},
		{
			name: "valid HTML creative",
			creative: AdvertisementCreative{
				Name:    "Test Creative",
				Type:    "html",
				Content: "<div>Test Ad</div>",
			},
			wantErr: false,
		},
		{
			name: "valid script creative",
			creative: AdvertisementCreative{
				Name:    "Test Creative",
				Type:    "script",
				Content: "console.log('test');",
			},
			wantErr: false,
		},
		{
			name: "valid video creative",
			creative: AdvertisementCreative{
				Name:    "Test Creative",
				Type:    "video",
				Content: "https://example.com/video.mp4",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			creative: AdvertisementCreative{
				Type:    "image",
				Content: "https://example.com/image.jpg",
			},
			wantErr: true,
		},
		{
			name: "missing content",
			creative: AdvertisementCreative{
				Name: "Test Creative",
				Type: "image",
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			creative: AdvertisementCreative{
				Name:    "Test Creative",
				Type:    "invalid",
				Content: "test content",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.creative.IsValidCreative()
			if (err != nil) != tt.wantErr {
				t.Errorf("AdvertisementCreative.IsValidCreative() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		name   string
		slice  []string
		item   string
		want   bool
	}{
		{
			name:  "item exists",
			slice: []string{"apple", "banana", "cherry"},
			item:  "banana",
			want:  true,
		},
		{
			name:  "item does not exist",
			slice: []string{"apple", "banana", "cherry"},
			item:  "grape",
			want:  false,
		},
		{
			name:  "empty slice",
			slice: []string{},
			item:  "apple",
			want:  false,
		},
		{
			name:  "single item match",
			slice: []string{"apple"},
			item:  "apple",
			want:  true,
		},
		{
			name:  "single item no match",
			slice: []string{"apple"},
			item:  "banana",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsString(tt.slice, tt.item)
			if got != tt.want {
				t.Errorf("containsString() = %v, want %v", got, tt.want)
			}
		})
	}
}