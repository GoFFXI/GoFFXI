package database

import (
	"context"
	"database/sql"
	"errors"
)

const (
	ConstraintCharacterJobsCharacterIDUnique = "character_jobs_character_id_unique"
)

type CharacterJobs struct {
	CharacterID uint32 `bun:"type:int unsigned,unique,pk"`
	WAR         uint8  `bun:"type:tinyint unsigned,notnull,default:1"`
	MNK         uint8  `bun:"type:tinyint unsigned,notnull,default:1"`
	WHM         uint8  `bun:"type:tinyint unsigned,notnull,default:1"`
	BLM         uint8  `bun:"type:tinyint unsigned,notnull,default:1"`
	RDM         uint8  `bun:"type:tinyint unsigned,notnull,default:1"`
	THF         uint8  `bun:"type:tinyint unsigned,notnull,default:1"`
	PLD         uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	DRK         uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	BST         uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	BRD         uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	RNG         uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	SAM         uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	NIN         uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	DRG         uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	SMN         uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	BLU         uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	COR         uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	PUP         uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	DNC         uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	SCH         uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	GEO         uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	RUN         uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
}

type CharacterJobsQueries interface {
	GetCharacterJobsByID(ctx context.Context, characterID uint32) (CharacterJobs, error)
	CreateCharacterJobs(ctx context.Context, characterJobs *CharacterJobs) (CharacterJobs, error)
	UpdateCharacterJobs(ctx context.Context, characterJobs *CharacterJobs) (CharacterJobs, error)
	DeleteCharacterJobs(ctx context.Context, characterID uint32) error
}

func (q *queriesImpl) GetCharacterJobsByID(ctx context.Context, characterID uint32) (CharacterJobs, error) {
	var characterJobs CharacterJobs

	err := q.db.NewSelect().Model(&characterJobs).Where("character_id = ?", characterID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CharacterJobs{}, ErrNotFound
		}

		return CharacterJobs{}, err
	}

	return characterJobs, nil
}

func (q *queriesImpl) CreateCharacterJobs(ctx context.Context, characterJobs *CharacterJobs) (CharacterJobs, error) {
	_, err := q.db.NewInsert().Model(characterJobs).Exec(ctx)
	if err != nil {
		if isViolationOfConstraint(err, ConstraintCharacterJobsCharacterIDUnique) {
			return CharacterJobs{}, ErrCharacterIDNotUnique
		}

		return CharacterJobs{}, err
	}

	return *characterJobs, nil
}

func (q *queriesImpl) UpdateCharacterJobs(ctx context.Context, characterJobs *CharacterJobs) (CharacterJobs, error) {
	_, err := q.db.NewUpdate().Model(characterJobs).WherePK().Exec(ctx)
	if err != nil {
		return CharacterJobs{}, err
	}

	return *characterJobs, nil
}

func (q *queriesImpl) DeleteCharacterJobs(ctx context.Context, characterID uint32) error {
	_, err := q.db.NewDelete().Model((*CharacterJobs)(nil)).Where("character_id = ?", characterID).Exec(ctx)
	return err
}
