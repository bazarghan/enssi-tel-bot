package seeder

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/persistence/postgres"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// Seeder holds the dependencies for the database seeding process.
type Seeder struct {
	DB *gorm.DB
}

// Config represents the structure of our seeder.config.yaml file.
type Config struct {
	Achievements []AchievementConfig `mapstructure:"achievements"`
	Courses      []CourseConfig      `mapstructure:"courses"`
}

type AchievementConfig struct {
	Title           string `mapstructure:"title"`
	Description     string `mapstructure:"description"`
	Type            string `mapstructure:"type"`
	ImageURL        string `mapstructure:"image_url"`
	MinWordRequired uint   `mapstructure:"min_word_required"`
	TotalItems      uint   `mapstructure:"total_items"`
	CellWidth       uint   `mapstructure:"cell_width"`
	CellHeight      uint   `mapstructure:"cell_height"`
}

type CourseConfig struct {
	Title                  string `mapstructure:"title"`
	PersianTitle           string `mapstructure:"persian_title"`
	Description            string `mapstructure:"description"`
	PersianDescription     string `mapstructure:"persian_description"`
	JSONFilePath           string `mapstructure:"json_file_path"`
	LinkedAchievementTitle string `mapstructure:"linked_achievement_title"`
}

// NewSeeder creates a new instance of the Seeder.
func NewSeeder(db *gorm.DB) *Seeder {
	return &Seeder{DB: db}
}

// Run executes the entire seeding process.
func (s *Seeder) Run() error {
	cfg, err := loadSeederConfig("./configs")
	if err != nil {
		return fmt.Errorf("failed to load seeder config: %w", err)
	}

	fmt.Println("Seeding achievements...")
	createdAchievements, err := s.seedAchievements(cfg.Achievements)
	if err != nil {
		return fmt.Errorf("failed to seed achievements: %w", err)
	}
	fmt.Println("Achievements seeded successfully.")

	fmt.Println("Seeding courses...")
	if err := s.seedCourses(cfg.Courses, createdAchievements); err != nil {
		return fmt.Errorf("failed to seed courses: %w", err)
	}
	fmt.Println("Courses seeded successfully.")

	return nil
}

// seedAchievements populates the achievements table from the config.
func (s *Seeder) seedAchievements(achConfigs []AchievementConfig) (map[string]postgres.AchievementModel, error) {
	createdMap := make(map[string]postgres.AchievementModel)
	for _, ac := range achConfigs {
		model := postgres.AchievementModel{
			Title:           ac.Title,
			Description:     ac.Description,
			Type:            ac.Type,
			ImageURL:        ac.ImageURL,
			MinWordRequired: ac.MinWordRequired,
			TotalItems:      ac.TotalItems,
			CellWidth:       ac.CellWidth,
			CellHeight:      ac.CellHeight,
		}
		// Use FirstOrCreate to prevent duplicates on re-running the seeder
		if err := s.DB.Where(postgres.AchievementModel{Title: ac.Title}).FirstOrCreate(&model).Error; err != nil {
			return nil, err
		}
		createdMap[model.Title] = model
	}
	return createdMap, nil
}

// seedCourses populates the courses table and all related word data.
func (s *Seeder) seedCourses(courseConfigs []CourseConfig, achievements map[string]postgres.AchievementModel) error {
	for _, cc := range courseConfigs {
		fmt.Printf("Processing course: %s\n", cc.Title)

		// Find the linked achievement
		linkedAch, ok := achievements[cc.LinkedAchievementTitle]
		if !ok {
			fmt.Printf("Warning: Could not find linked achievement '%s' for course '%s'. Setting ID to 0.\n", cc.LinkedAchievementTitle, cc.Title)
		}

		courseModel := postgres.CourseModel{
			Title:               cc.Title,
			PersianTitle:        cc.PersianTitle,
			Description:         cc.Description,
			PersianDescription:  cc.PersianDescription,
			LinkedAchievementID: linkedAch.ID, // Will be 0 if not found, which is safe
		}

		if err := s.DB.Where(postgres.CourseModel{Title: cc.Title}).FirstOrCreate(&courseModel).Error; err != nil {
			return err
		}

		// Check if course already has words to avoid re-populating
		var wordCount int64
		s.DB.Model(&postgres.CourseWordModel{}).Where("course_id = ?", courseModel.ID).Count(&wordCount)
		if wordCount > 0 {
			fmt.Printf("Course '%s' already has %d words. Skipping population.\n", cc.Title, wordCount)
			continue
		}

		if err := s.populateCourseFromJSON(cc.JSONFilePath, &courseModel); err != nil {
			return fmt.Errorf("error populating course '%s' from JSON: %w", cc.Title, err)
		}
		fmt.Printf("Finished processing course: %s\n", cc.Title)
	}
	return nil
}

// populateCourseFromJSON reads a JSON file and populates words for a specific course.
func (s *Seeder) populateCourseFromJSON(filePath string, course *postgres.CourseModel) error {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	var orderedList []OrderedWordListItem
	if err := json.Unmarshal(fileBytes, &orderedList); err != nil {
		return fmt.Errorf("error unmarshaling json from %s: %w", filePath, err)
	}

	// Use a transaction for performance
	return s.DB.Transaction(func(tx *gorm.DB) error {
		for i, item := range orderedList {
			if (i+1)%100 == 0 {
				fmt.Printf("...populating word %d/%d for course '%s'\n", i+1, len(orderedList), course.Title)
			}

			// Using FirstOrCreate for idempotency
			source := postgres.SourceModel{Title: "Vocabulary.com", URL: "vocabulary.com"}
			tx.Where(postgres.SourceModel{URL: source.URL}).FirstOrCreate(&source)

			word := postgres.WordModel{Lang: "En", Title: item.Word}
			tx.Where(postgres.WordModel{Title: item.Word, Lang: "En"}).FirstOrCreate(&word)

			courseWord := postgres.CourseWordModel{
				CourseID:          course.ID,
				WordID:            word.ID,
				TelgramImageID:    item.Details.FlashCard.TelgramImageID,
				TelgramImageDocID: item.Details.FlashCard.TelgramImageDocID,
				Lesson:            fmt.Sprintf("Unit %d", (i+11)/12),
				Index:             uint(i + 1),
			}
			tx.Create(&courseWord)

			wordSource := postgres.WordSourceModel{
				WordID:       word.ID,
				SourceID:     source.ID,
				DefPrimary:   item.Details.DefPrimary,
				DefSecondary: item.Details.DefSecondary,
			}
			tx.Create(&wordSource)

			for _, posData := range item.Details.PartOfSpeeches {
				ps := postgres.PartOfSpeechModel{WordSourceID: wordSource.ID, Title: posData.Title}
				tx.Create(&ps)
				for _, meaningData := range posData.Meanings {
					m := postgres.MeaningModel{PartOfSpeechID: ps.ID, Lang: meaningData.Lang, Title: meaningData.Title}
					tx.Create(&m)
				}
			}

			for _, pronData := range item.Details.Pronunciations {
				if pronData.Region == "US" {
					p := postgres.PronunciationModel{
						WordSourceID:   wordSource.ID,
						Region:         pronData.Region,
						URL:            pronData.URL,
						TelgramVoiceID: item.Details.TelgramVoiceDocID,
					}
					tx.Create(&p)
				}
			}

			for _, phoneticData := range item.Details.Phonetics {
				ph := postgres.PhoneticModel{WordSourceID: wordSource.ID, Lang: phoneticData.Lang, Title: phoneticData.Title}
				tx.Create(&ph)
			}
		}
		return nil
	})
}

// loadSeederConfig reads configuration from the seeder.config.yaml file.
func loadSeederConfig(path string) (*Config, error) {
	v := viper.New()
	v.AddConfigPath(path)
	v.SetConfigName("seeder.config")
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read seeder config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal seeder config: %w", err)
	}

	return &cfg, nil
}
