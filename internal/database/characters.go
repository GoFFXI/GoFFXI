package database

import (
	"context"
	"database/sql"
	"errors"
)

const (
	ConstraintCharactersNameUnique = "characters_name_unique"
	ConstraintCharactersIDUnique   = "characters_id_unique"
)

var ErrCharacterNameNotUnique = errors.New("character name not unique")

type Character struct {
	ID                uint32 `bun:"type:int unsigned,unique,pk,autoincrement"`
	AccountID         uint32 `bun:"type:int unsigned"`
	OriginalAccountID uint32 `bun:"type:int unsigned"`
	Name              string `bun:"type:varchar(16),notnull,unique"`
	Nation            uint8  `bun:"type:tinyint unsigned,notnull"`
	PosZone           uint16 `bun:"type:smallint unsigned,notnull"`

	Jobs  *CharacterJobs  `bun:"rel:has-one,join:id=character_id"`
	Stats *CharacterStats `bun:"rel:has-one,join:id=character_id"`
	Looks *CharacterLooks `bun:"rel:has-one,join:id=character_id"`
}

func (c *Character) GetMainJobLevel() uint8 {
	if c.Jobs == nil || c.Stats == nil {
		return 0
	}

	// Create array where index = job constant, value = job level
	jobLevels := []uint8{
		0,          // JobNone (0)
		c.Jobs.WAR, // JobWarrior (1)
		c.Jobs.MNK, // JobMonk (2)
		c.Jobs.WHM, // JobWhiteMage (3)
		c.Jobs.BLM, // JobBlackMage (4)
		c.Jobs.RDM, // JobRedMage (5)
		c.Jobs.THF, // JobThief (6)
		c.Jobs.PLD, // JobPaladin (7)
		c.Jobs.DRK, // JobDarkKnight (8)
		c.Jobs.BST, // JobBeastmaster (9)
		c.Jobs.BRD, // JobBard (10)
		c.Jobs.RNG, // JobRanger (11)
		c.Jobs.SAM, // JobSamurai (12)
		c.Jobs.NIN, // JobNinja (13)
		c.Jobs.DRG, // JobDragoon (14)
		c.Jobs.SMN, // JobSummoner (15)
		c.Jobs.BLU, // JobBlueMage (16)
		c.Jobs.COR, // JobCorsair (17)
		c.Jobs.PUP, // JobPuppetmaster (18)
		c.Jobs.DNC, // JobDancer (19)
		c.Jobs.SCH, // JobScholar (20)
		c.Jobs.GEO, // JobGeomancer (21)
		c.Jobs.RUN, // JobRuneFencer (22)
	}

	//nolint:gosec // the length will never overflow
	if c.Stats.MainJob < uint8(len(jobLevels)) {
		return jobLevels[c.Stats.MainJob]
	}

	return 0
}

type CharacterQueries interface {
	GetCharacterByID(ctx context.Context, characterID uint32) (Character, error)
	GetCharactersByAccountID(ctx context.Context, accountID uint32) ([]Character, error)
	CountCharactersByAccountID(ctx context.Context, accountID uint32) (int, error)
	CreateCharacter(ctx context.Context, character *Character) (Character, error)
	UpdateCharacter(ctx context.Context, character *Character) (Character, error)
	DeleteCharacter(ctx context.Context, characterID uint32) error
	CharacterNameExists(ctx context.Context, characterName string) (bool, error)
}

func (q *queriesImpl) GetCharacterByID(ctx context.Context, characterID uint32) (Character, error) {
	var character Character

	err := q.db.NewSelect().Model(&character).Where("id = ?", characterID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Character{}, ErrNotFound
		}

		return Character{}, err
	}

	return character, nil
}

func (q *queriesImpl) GetCharactersByAccountID(ctx context.Context, accountID uint32) ([]Character, error) {
	var characters []Character

	err := q.db.NewSelect().Model(&characters).Where("account_id = ?", accountID).Order("id ASC").Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return characters, nil
		}

		return nil, err
	}

	return characters, nil
}

func (q *queriesImpl) CountCharactersByAccountID(ctx context.Context, accountID uint32) (int, error) {
	count, err := q.db.NewSelect().Model((*Character)(nil)).Where("account_id = ?", accountID).Count(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}

		return 0, err
	}

	return count, nil
}

func (q *queriesImpl) CreateCharacter(ctx context.Context, character *Character) (Character, error) {
	_, err := q.db.NewInsert().Model(character).Exec(ctx)
	if err != nil {
		if isViolationOfConstraint(err, ConstraintCharactersNameUnique) {
			return Character{}, ErrCharacterNameNotUnique
		}

		if isViolationOfConstraint(err, ConstraintCharactersIDUnique) {
			return Character{}, ErrCharacterIDNotUnique
		}

		return Character{}, err
	}

	return *character, nil
}

func (q *queriesImpl) UpdateCharacter(ctx context.Context, character *Character) (Character, error) {
	_, err := q.db.NewUpdate().Model(character).WherePK().Exec(ctx)
	if err != nil {
		if isViolationOfConstraint(err, ConstraintCharactersNameUnique) {
			return Character{}, ErrCharacterNameNotUnique
		}

		return Character{}, err
	}

	return *character, nil
}

func (q *queriesImpl) DeleteCharacter(ctx context.Context, characterID uint32) error {
	_, err := q.db.NewDelete().Model((*Character)(nil)).Where("id = ?", characterID).Exec(ctx)
	return err
}

func (q *queriesImpl) CharacterNameExists(ctx context.Context, characterName string) (bool, error) {
	count, err := q.db.NewSelect().Model((*Character)(nil)).Where("name = ?", characterName).Count(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return count > 0, nil
}
