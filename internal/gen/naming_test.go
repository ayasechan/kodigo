package gen

import "testing"

func TestExportName(t *testing.T) {
	cases := map[string]string{
		"Player.Repeat": "PlayerRepeat",
		"songid":        "SongID",
		"movieid":       "MovieID",
		"channeluid":    "ChannelUID",
		"tvshowid":      "TVshowID", // no separator: compound stays joined
		"showtitle":     "Showtitle",
		"fanart":        "Fanart",
		"url":           "URL",
		"type":          "Type",
		"playerid":      "PlayerID",
	}
	for in, want := range cases {
		if got := exportName(in); got != want {
			t.Errorf("exportName(%q) = %q, want %q", in, got, want)
		}
	}
	if got := fieldName("id"); got != "ID" {
		t.Errorf("fieldName(id) = %q, want ID", got)
	}
	if got := enumConstName("PlayerRepeat", "off"); got != "PlayerRepeatOff" {
		t.Errorf("enumConstName = %q, want PlayerRepeatOff", got)
	}
}
