package postgres

import (
	"database/sql/driver"
	"fmt"
	"github.com/bits-and-blooms/bitset"
	"gorm.io/gorm"
)

// GormBitSet implements driver.Valuer and sql.Scanner for a bit-string column.
type GormBitSet struct {
	bitset.BitSet
}

func (b GormBitSet) Value() (driver.Value, error) {

	return b.BitSet.MarshalBinary()
}

func (b *GormBitSet) Scan(value interface{}) error {
	if value == nil {
		b.BitSet = *bitset.New(0)
		return nil
	}
	bv, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan BitSet: unsupported type %T", value)
	}
	newBs := &bitset.BitSet{}
	if err := newBs.UnmarshalBinary(bv); err != nil {
		return fmt.Errorf("failed to unmarshal binary to BitSet: %w", err)
	}
	b.BitSet = *newBs
	return nil
}

func (GormBitSet) GormDataType() string {
	return "BYTEA"
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
	UserID        uint
	AchievementID uint
	State         GormBitSet       // Custom type for bitset storage
	Achievement   achievementModel `gorm:"foreignKey:AchievementID"`
}

func (userAchievementModel) TableName() string { return "user_achievements" }
