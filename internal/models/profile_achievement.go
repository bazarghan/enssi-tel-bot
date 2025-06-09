package models

import (
	"database/sql/driver"
	"fmt"
	"strings" // Import for strings.Builder

	"github.com/bits-and-blooms/bitset"
	"gorm.io/gorm"
)

// GormBitSet implements driver.Valuer and sql.Scanner for a bit-string column.
type GormBitSet struct {
	bitset.BitSet
	TotalLength uint
}

// Value serialises the in-memory bitset into the Postgres BIT literal form B'0101'.
func (b GormBitSet) Value() (driver.Value, error) {
	// Choose the declared total length if present; fall back to the real length.
	length := b.TotalLength
	if length == 0 {
		length = b.Len()
	}
	if length == 0 {
		return "B''", nil
	}

	var sb strings.Builder
	for i := uint(0); i < length; i++ {
		if b.Test(i) {
			sb.WriteByte('1')
		} else {
			sb.WriteByte('0')
		}
	}
	return fmt.Sprintf("B'%s'", sb.String()), nil
}

// Scan deserialises a BIT or VARBIT value back into the in-memory bitset.
func (b *GormBitSet) Scan(value interface{}) error {
	if value == nil {
		*b = GormBitSet{BitSet: *bitset.New(0)}
		return nil
	}

	var raw string
	switch v := value.(type) {
	case []byte:
		raw = string(v)
	case string:
		raw = v
	default:
		return fmt.Errorf("failed to scan BitSet: unsupported type %T", value)
	}

	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "B''" || raw == "b''" {
		*b = GormBitSet{BitSet: *bitset.New(0)}
		return nil
	}

	// Accept both Postgres literal B'0101' and plain 0101.
	if n := len(raw); n >= 3 &&
		(raw[0] == 'B' || raw[0] == 'b') &&
		raw[1] == '\'' && raw[n-1] == '\'' {
		raw = raw[2 : n-1]
	}

	newBS := bitset.New(uint(len(raw)))
	for i, ch := range raw {
		if ch == '1' {
			newBS.Set(uint(i))
		} else if ch != '0' {
			return fmt.Errorf("invalid character %q in bit string", ch)
		}
	}

	b.BitSet = *newBS
	b.TotalLength = uint(len(raw))
	return nil
}

// GormDataType returns the GORM data type for GormBitSet
func (GormBitSet) GormDataType() string {
	return "BIT VARYING"
}

type ProfileAchievement struct {
	gorm.Model
	ProfileID     uint
	AchievementID uint
	State         GormBitSet // Your custom payload data

	// Belongs To relationships for preloading
	Profile     Profile     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Achievement Achievement `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
