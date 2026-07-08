package normalize

// volumeMultiplier maps adapter names to their volume-to-shares multiplier.
// A-share data sources return volume in 手 (lots of 100 shares).
var volumeMultiplier = map[string]float64{
	"eastmoney": 100,
	"sina":      100,
	"tencent":   100,
	"tushare":   100,
	"mootdx":    100,
	"baidu":     100,
}

// NormalizeVolume converts trading volume to standard shares.
// Returns volume unchanged for unknown or non-A-share sources.
func NormalizeVolume(source string, volume float64) float64 {
	if mult, ok := volumeMultiplier[source]; ok {
		return volume * mult
	}
	return volume
}

// VolumeMultiplier returns the volume multiplier for a source, or 1 for unknown sources.
func VolumeMultiplier(source string) float64 {
	if mult, ok := volumeMultiplier[source]; ok {
		return mult
	}
	return 1
}

// VolumeSources returns the list of known A-share data sources.
func VolumeSources() []string {
	sources := make([]string, 0, len(volumeMultiplier))
	for s := range volumeMultiplier {
		sources = append(sources, s)
	}
	return sources
}
