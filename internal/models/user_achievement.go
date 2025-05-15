package models

import (
	"database/sql/driver"
	"fmt"
	"strings" // Import for strings.Builder

	"github.com/bits-and-blooms/bitset"
	"gorm.io/gorm"
)

// GormBitSet is a wrapper around bitset.BitSet to implement GORM serialization
type GormBitSet struct {
	bitset.BitSet
}

// Value implements the driver.Valuer interface for GormBitSet.
// This converts the GormBitSet to a string representation suitable for PostgreSQL's BIT VARYING.
// PostgreSQL expects bit strings in the format B'101010'.
func (b GormBitSet) Value() (driver.Value, error) {
	// Corrected check for empty bitset
	if b.Count() == 0 { // Use Count() to check if any bits are set
		return "B''", nil // Represents an empty bit string in PostgreSQL
	}

	var sb strings.Builder
	// Len() returns the "logical length", which is the highest set bit + 1.
	// We need to iterate up to this length to capture all bits accurately.
	length := b.Len()
	for i := uint(0); i < length; i++ {
		if b.Test(i) {
			sb.WriteString("1")
		} else {
			sb.WriteString("0")
		}
	}
	return fmt.Sprintf("B'%s'", sb.String()), nil
}

// Scan implements the sql.Scanner interface for GormBitSet.
// This converts data from PostgreSQL (expected to be a string representation of a bitmask)
// back into a GormBitSet.
func (b *GormBitSet) Scan(value interface{}) error {
	if value == nil {
		// Initialize to an empty bitset
		b.BitSet = *bitset.New(0)
		return nil
	}

	var byteSlice []byte
	switch v := value.(type) {
	case []byte:
		byteSlice = v
	case string:
		byteSlice = []byte(v)
	default:
		return fmt.Errorf("failed to scan BitSet: unsupported type %T", value)
	}

	// The string from PostgreSQL for BIT VARYING will be like "101010"
	// (without the B'' prefix when read back directly by some drivers).
	strVal := string(byteSlice)

	// Handle the case where PostgreSQL might return B'' for an empty bit string.
	// Some drivers might return an empty string "" directly.
	if strVal == "" || strVal == "B''" { // Added check for B'' just in case
		b.BitSet = *bitset.New(0)
		return nil
	}

	// If strVal has B' prefix from some specific driver behavior (less common on direct read)
	// we might need to strip it. For now, assuming direct bit string like "1010".
	// Example: if strings.HasPrefix(strVal, "B'") && strings.HasSuffix(strVal, "'") {
	//     strVal = strVal[2 : len(strVal)-1]
	// }

	newBitSet := bitset.New(uint(len(strVal)))
	for i, r := range strVal {
		if r == '1' {
			newBitSet.Set(uint(i))
		} else if r == '0' {
			// Do nothing, already 0
		} else {
			return fmt.Errorf("failed to scan BitSet: invalid character '%c' in bit string", r)
		}
	}
	b.BitSet = *newBitSet
	return nil
}

// GormDataType returns the GORM data type for GormBitSet
func (GormBitSet) GormDataType() string {
	return "BIT VARYING"
}

// UserAchievement model using the custom GormBitSet type
type UserAchievement struct {
	gorm.Model
	ProfileID     uint
	AchievementID uint
	State         GormBitSet // Use the custom type
}
