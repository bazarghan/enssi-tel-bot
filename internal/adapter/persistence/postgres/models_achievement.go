package postgres

import (
	"database/sql/driver"
	"fmt"
	"github.com/bits-and-blooms/bitset"
	"gorm.io/gorm"
	"strings"
	"time"
)

// GormBitSet now includes TotalLength to ensure correct serialization.
type GormBitSet struct {
	bitset.BitSet
	TotalLength uint
}

// Value implements the driver.Valuer interface for GormBitSet.
func (b GormBitSet) Value() (driver.Value, error) {
	// --- START OF FIX ---
	// This is the corrected implementation.

	length := b.TotalLength
	if length == 0 && b.BitSet.Len() > 0 {
		length = b.BitSet.Len()
	}

	if length == 0 {
		// Return an empty string. The database driver will handle it correctly.
		return "", nil
	}

	var sb strings.Builder
	for i := uint(0); i < length; i++ {
		if b.Test(i) {
			sb.WriteString("1")
		} else {
			sb.WriteString("0")
		}
	}

	// We now return the RAW bit string (e.g., "111000") without any "B'" prefix.
	// The database driver is responsible for formatting this correctly.
	return sb.String(), nil
	// --- END OF FIX ---
}

// Scan method is unchanged and correct as it infers length from the input string.
// In internal/adapter/persistence/postgres/models_achievement.go

// Scan implements the sql.Scanner interface for GormBitSet.
func (b *GormBitSet) Scan(value interface{}) error {
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
	if strVal == "" {
		b.BitSet = *bitset.New(0)
		b.TotalLength = 0
		return nil
	}

	// --- START OF FIX ---
	// This is the critical change. We must trim the "B'" prefix and the final "'"
	// that PostgreSQL adds to the BIT VARYING type.
	if len(strVal) > 3 && strings.HasPrefix(strVal, "B'") && strings.HasSuffix(strVal, "'") {
		strVal = strVal[2 : len(strVal)-1]
	} else if len(strVal) > 2 && strVal[0] == 'B' { // Handle cases like B'101'
		strVal = strVal[1:]
	}
	// This handles the case where the database just returns the bits, e.g., "10101".
	// After this block, strVal will be ONLY "1"s and "0"s.
	// --- END OF FIX ---

	newBitSet := bitset.New(uint(len(strVal)))
	for i, r := range strVal {
		if r == '1' {
			newBitSet.Set(uint(i))
		}
	}
	b.BitSet = *newBitSet
	b.TotalLength = uint(len(strVal))
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
	CellWidth       uint
	CellHeight      uint
}

func (achievementModel) TableName() string { return "achievements" }

// userAchievementModel is the GORM struct for the join table.
type userAchievementModel struct {
	gorm.Model
	ProfileID     uint `gorm:"column:profile_id"`
	AchievementID uint
	State         GormBitSet       // Custom type for bitset storage
	Achievement   achievementModel `gorm:"foreignKey:AchievementID"`
	CompletedAt   *time.Time       `gorm:"null"`
}

func (userAchievementModel) TableName() string { return "profile_achievements" }
