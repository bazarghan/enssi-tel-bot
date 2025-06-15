package postgres

import (
	"database/sql/driver"
	"fmt"
	"github.com/bits-and-blooms/bitset"
	"gorm.io/gorm"
	"strings"
)

// GormBitSet now includes TotalLength to ensure correct serialization.
type GormBitSet struct {
	bitset.BitSet
	TotalLength uint
}

// Value implements the driver.Valuer interface for GormBitSet.
func (b GormBitSet) Value() (driver.Value, error) {
	// --- THIS IS THE FIX ---
	// Use the explicit TotalLength instead of b.Len() to serialize the full bitmask.
	// If TotalLength is not set (e.g., for older records), fall back to b.Len()
	// to avoid saving a zero-length string for existing data.
	length := b.TotalLength
	if length == 0 && b.Len() > 0 {
		length = b.Len()
	}

	if length == 0 {
		return "B''", nil
	}

	var sb strings.Builder
	for i := uint(0); i < length; i++ {
		if b.Test(i) {
			sb.WriteString("1")
		} else {
			sb.WriteString("0")
		}
	}
	return fmt.Sprintf("B'%s'", sb.String()), nil
	// --- END OF FIX ---
}

// Scan method is unchanged and correct as it infers length from the input string.
func (b *GormBitSet) Scan(value interface{}) error {
	// ... (existing Scan logic is correct) ...
	// When scanning, we also need to set the TotalLength.
	if value == nil {
		b.BitSet = *bitset.New(0)
		b.TotalLength = 0
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
	strVal := string(byteSlice)
	if strVal == "" || strVal == "B''" {
		b.BitSet = *bitset.New(0)
		b.TotalLength = 0
		return nil
	}
	newBitSet := bitset.New(uint(len(strVal)))
	for i, r := range strVal {
		if r == '1' {
			newBitSet.Set(uint(i))
		}
	}
	b.BitSet = *newBitSet
	b.TotalLength = uint(len(strVal)) // Set TotalLength based on what was read from DB
	return nil
}

// GormDataType returns the GORM data type for GormBitSet
func (GormBitSet) GormDataType() string {
	return "BIT VARYING"
}

// achievementModel is the GORM struct for the 'achievements' table.
type achievementModel struct {
	gorm.Model
	Title           string `gorm:"not null;unique"`
	Description     string
	Type            string
	ImageURL        string
	MinWordRequired uint
	TotalItems      uint
	GridWidth       uint
	GridHeight      uint
}

func (achievementModel) TableName() string { return "achievements" }

// userAchievementModel is the GORM struct for the join table.
type userAchievementModel struct {
	gorm.Model
	ProfileID     uint `gorm:"column:profile_id"`
	AchievementID uint
	State         GormBitSet       // Custom type for bitset storage
	Achievement   achievementModel `gorm:"foreignKey:AchievementID"`
}

func (userAchievementModel) TableName() string { return "profile_achievements" }
