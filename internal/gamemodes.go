package internal

type GameMode uint16

const (
	ModeClassic GameMode = 1
	ModeRanked  GameMode = 2
	ModeCasual  GameMode = 3
	ModeCustom  GameMode = 4
)

func (m GameMode) String() string {
	if name, ok := ModeNames[m]; ok {
		return name
	}
	return "Unknown"
}

var ModeNames = map[GameMode]string{
	ModeClassic: "Classic",
	ModeRanked:  "Ranked",
	ModeCasual:  "Casual",
	ModeCustom:  "Custom",
}

var AvailableModes = []GameMode{
	ModeClassic,
	ModeRanked,
	ModeCasual,
	ModeCustom,
}

func GetAllModes() []uint16 {
	modes := make([]uint16, len(AvailableModes))
	for i, mode := range AvailableModes {
		modes[i] = uint16(mode)
	}
	return modes
}
