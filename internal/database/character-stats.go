package database

import (
	"context"
	"database/sql"
	"errors"
)

const (
	ConstraintCharacterStatsCharacterIDUnique = "character_stats_character_id_unique"
)

type CharacterStats struct {
	CharacterID uint32 `bun:"type:int unsigned,unique,pk"`
	HP          uint16 `bun:"type:smallint unsigned,notnull,default:50"`
	MP          uint16 `bun:"type:smallint unsigned,notnull,default:50"`
	MainJob     uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	SubJob      uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
}

type CharacterStatsQueries interface {
	GetCharacterStatsByID(ctx context.Context, characterID uint32) (CharacterStats, error)
	CreateCharacterStats(ctx context.Context, characterStats *CharacterStats) (CharacterStats, error)
	UpdateCharacterStats(ctx context.Context, characterStats *CharacterStats) (CharacterStats, error)
	DeleteCharacterStats(ctx context.Context, characterID uint32) error
}

func (q *queriesImpl) GetCharacterStatsByID(ctx context.Context, characterID uint32) (CharacterStats, error) {
	var characterStats CharacterStats

	err := q.db.NewSelect().Model(&characterStats).Where("character_id = ?", characterID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CharacterStats{}, ErrNotFound
		}

		return CharacterStats{}, err
	}

	return characterStats, nil
}

func (q *queriesImpl) CreateCharacterStats(ctx context.Context, characterStats *CharacterStats) (CharacterStats, error) {
	_, err := q.db.NewInsert().Model(characterStats).Exec(ctx)
	if err != nil {
		if isViolationOfConstraint(err, ConstraintCharacterStatsCharacterIDUnique) {
			return CharacterStats{}, ErrCharacterIDNotUnique
		}

		return CharacterStats{}, err
	}

	return *characterStats, nil
}

func (q *queriesImpl) UpdateCharacterStats(ctx context.Context, characterStats *CharacterStats) (CharacterStats, error) {
	_, err := q.db.NewUpdate().Model(characterStats).Where("character_id = ?", characterStats.CharacterID).Exec(ctx)
	if err != nil {
		return CharacterStats{}, err
	}

	return *characterStats, nil
}

func (q *queriesImpl) DeleteCharacterStats(ctx context.Context, characterID uint32) error {
	_, err := q.db.NewDelete().Model((*CharacterStats)(nil)).Where("character_id = ?", characterID).Exec(ctx)
	return err
}
