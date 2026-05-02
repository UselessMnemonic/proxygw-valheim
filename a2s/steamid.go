package a2s

type Universe uint8

const (
	UniversePublic Universe = 1
)

type AccountType uint8

const (
	AccountTypeAnonGameServer AccountType = 4
)

func SteamID(universe Universe, accountType AccountType, instance uint32, accountID uint32) uint64 {
	return uint64(accountID) |
		(uint64(instance&0x000fffff) << 32) |
		(uint64(accountType&0x0f) << 52) |
		(uint64(universe) << 56)
}
