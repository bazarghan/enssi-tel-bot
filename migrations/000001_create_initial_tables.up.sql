-- Enable the citext extension for case-insensitive text
CREATE EXTENSION IF NOT EXISTS citext;

-- User and Profile Tables
CREATE TABLE "users" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "telegram_id" BIGINT NOT NULL UNIQUE,
    "last_active" TIMESTAMPTZ,
    "last_online" TIMESTAMPTZ,
    "last_menu" VARCHAR(100) NOT NULL DEFAULT '',
    "last_review_session_completed_at" TIMESTAMPTZ,
    "is_admin" BOOLEAN DEFAULT FALSE
);

CREATE TABLE "profiles" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "user_id" BIGINT REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    "username" TEXT,
    "first_name" TEXT,
    "last_name" TEXT,
    "date_of_birth" TIMESTAMPTZ,
    "score" BIGINT
);

-- Achievement Tables
CREATE TABLE "achievements" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "title" TEXT NOT NULL UNIQUE,
    "description" TEXT,
    "type" TEXT,
    "image_url" TEXT,
    "min_word_required" BIGINT,
    "total_items" BIGINT,
    "cell_width" BIGINT,
    "cell_height" BIGINT
);

CREATE TABLE "profile_achievements" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "profile_id" BIGINT REFERENCES "profiles"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    "achievement_id" BIGINT REFERENCES "achievements"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    "state" BIT VARYING,
    "completed_at" TIMESTAMPTZ
);

-- Course and Word Tables
CREATE TABLE "courses" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "title" TEXT NOT NULL,
    "persian_title" TEXT NOT NULL,
    "description" TEXT,
    "persian_description" TEXT,
    "linked_achievement_id" BIGINT DEFAULT 0
);

CREATE TABLE "user_courses" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "user_id" BIGINT REFERENCES "users"("id") ON DELETE CASCADE,
    "course_id" BIGINT REFERENCES "courses"("id") ON DELETE CASCADE,
    "progress" BIGINT
);

CREATE TABLE "words" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "lang" TEXT NOT NULL,
    "title" TEXT NOT NULL
);

CREATE TABLE "course_words" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "course_id" BIGINT REFERENCES "courses"("id"),
    "word_id" BIGINT REFERENCES "words"("id"),
    "telgram_image_id" TEXT,
    "telgram_image_doc_id" TEXT,
    "lesson" TEXT,
    "index" BIGINT
);

-- Word Details Tables
CREATE TABLE "sources" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "title" TEXT NOT NULL,
    "description" TEXT,
    "url" TEXT NOT NULL
);

CREATE TABLE "word_sources" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "word_id" BIGINT REFERENCES "words"("id") ON DELETE CASCADE,
    "source_id" BIGINT REFERENCES "sources"("id"),
    "def_primary" TEXT,
    "def_secondary" TEXT
);

CREATE TABLE "pronunciations" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "word_source_id" BIGINT REFERENCES "word_sources"("id") ON DELETE CASCADE,
    "region" TEXT NOT NULL,
    "url" TEXT NOT NULL,
    "telgram_voice_id" TEXT
);

CREATE TABLE "images" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "word_source_id" BIGINT REFERENCES "word_sources"("id") ON DELETE CASCADE,
    "url" TEXT NOT NULL
);

CREATE TABLE "phonetics" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "word_source_id" BIGINT REFERENCES "word_sources"("id") ON DELETE CASCADE,
    "lang" TEXT NOT NULL,
    "title" TEXT NOT NULL
);

CREATE TABLE "part_of_speeches" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "word_source_id" BIGINT REFERENCES "word_sources"("id") ON DELETE CASCADE,
    "title" TEXT NOT NULL
);

CREATE TABLE "meanings" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "part_of_speech_id" BIGINT REFERENCES "part_of_speeches"("id") ON DELETE CASCADE,
    "lang" TEXT NOT NULL,
    "title" TEXT NOT NULL
);

-- SRS Tables (Spaced Repetition System)
CREATE TABLE "word_studieds" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "user_id" BIGINT REFERENCES "users"("id") ON DELETE CASCADE,
    "word_id" BIGINT REFERENCES "words"("id") ON DELETE CASCADE,
    "last_reviewed_at" TIMESTAMPTZ,
    "next_review_at" TIMESTAMPTZ,
    "review_interval_days" BIGINT DEFAULT 1,
    "is_mastered" BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX "idx_word_studieds_user_id" ON "word_studieds"("user_id");
CREATE INDEX "idx_word_studieds_word_id" ON "word_studieds"("word_id");
CREATE INDEX "idx_word_studieds_next_review_at" ON "word_studieds"("next_review_at");

CREATE TABLE "word_studied_todays" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "user_id" BIGINT REFERENCES "users"("id"),
    "word_id" BIGINT REFERENCES "words"("id"),
    "course_id" BIGINT REFERENCES "courses"("id"),
    "point" INTEGER
);

-- Quiz Tables
CREATE TABLE "quizzes" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "course_id" BIGINT,
    "question_count" BIGINT,
    "type" VARCHAR(50) DEFAULT 'COURSE_BLOCK',
    "trigger_progress" BIGINT
);

CREATE TABLE "quiz_questions" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "quiz_id" BIGINT REFERENCES "quizzes"("id") ON DELETE CASCADE,
    "text" TEXT,
    "word_id" BIGINT
);
CREATE INDEX "idx_quiz_questions_word_id" ON "quiz_questions"("word_id");

CREATE TABLE "quiz_question_options" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "quiz_question_id" BIGINT NOT NULL REFERENCES "quiz_questions"("id") ON DELETE CASCADE,
    "text" TEXT NOT NULL,
    "is_correct" BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE "quiz_attempts" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "quiz_id" BIGINT NOT NULL REFERENCES "quizzes"("id") ON DELETE CASCADE,
    "user_id" BIGINT NOT NULL REFERENCES "users"("id") ON DELETE CASCADE,
    "score" INTEGER DEFAULT 0,
    "is_completed" BOOLEAN DEFAULT FALSE,
    "current_question_num" INTEGER DEFAULT 0,
    "current_question_message_id" INTEGER
);

CREATE TABLE "quiz_answers" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "quiz_attempt_id" BIGINT REFERENCES "quiz_attempts"("id") ON DELETE CASCADE,
    "quiz_question_id" BIGINT REFERENCES "quiz_questions"("id") ON DELETE CASCADE,
    "quiz_question_option_id" BIGINT REFERENCES "quiz_question_options"("id") ON DELETE CASCADE,
    "is_correct" BOOLEAN
);
