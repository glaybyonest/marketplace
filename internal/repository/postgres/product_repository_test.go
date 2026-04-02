package postgres

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalProductMetadataKeepsPrimaryImageFirst(t *testing.T) {
	galleryRaw, specsRaw, err := marshalProductMetadata(
		" https://cdn.example.com/hero.png ",
		[]string{
			"https://cdn.example.com/detail.png",
			"https://cdn.example.com/hero.png",
			" ",
		},
		map[string]any{"weight": "1 kg"},
	)
	require.NoError(t, err)

	var gallery []string
	require.NoError(t, json.Unmarshal(galleryRaw, &gallery))
	assert.Equal(t, []string{
		"https://cdn.example.com/hero.png",
		"https://cdn.example.com/detail.png",
	}, gallery)

	var specs map[string]any
	require.NoError(t, json.Unmarshal(specsRaw, &specs))
	assert.Equal(t, map[string]any{"weight": "1 kg"}, specs)
}

func TestDecodeProductImagesDeduplicatesGallery(t *testing.T) {
	images, err := decodeProductImages([]byte(`[
		"https://cdn.example.com/hero.png",
		"https://cdn.example.com/detail.png",
		"https://cdn.example.com/hero.png"
	]`))
	require.NoError(t, err)
	assert.Equal(t, []string{
		"https://cdn.example.com/hero.png",
		"https://cdn.example.com/detail.png",
	}, images)
}
