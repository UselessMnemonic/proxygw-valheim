package a2s

type EDF byte

const (
	EDFGameID   EDF = 0x01
	EDFSteamID  EDF = 0x10
	EDFKeywords EDF = 0x20
	EDFPort     EDF = 0x80
)
