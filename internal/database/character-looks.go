package database

import (
	"context"
	"database/sql"
	"errors"
)

const (
	ConstraintCharacterLooksCharacterIDUnique = "character_looks_character_id_unique"
)

type CharacterLooks struct {
	CharacterID uint32 `bun:"type:int unsigned,unique,pk"`
	Face        uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	Race        uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	Size        uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	Head        uint16 `bun:"type:smallint unsigned,notnull,default:0"`
	Body        uint16 `bun:"type:smallint unsigned,notnull,default:8"`
	Hands       uint16 `bun:"type:smallint unsigned,notnull,default:8"`
	Legs        uint16 `bun:"type:smallint unsigned,notnull,default:8"`
	Feet        uint16 `bun:"type:smallint unsigned,notnull,default:8"`
	Main        uint16 `bun:"type:smallint unsigned,notnull,default:0"`
	Sub         uint16 `bun:"type:smallint unsigned,notnull,default:0"`
	Ranged      uint16 `bun:"type:smallint unsigned,notnull,default:0"`
}

type CharacterLooksQueries interface {
	GetCharacterLooksByID(ctx context.Context, characterID uint32) (CharacterLooks, error)
	CreateCharacterLooks(ctx context.Context, characterLooks *CharacterLooks) (CharacterLooks, error)
	UpdateCharacterLooks(ctx context.Context, characterLooks *CharacterLooks) (CharacterLooks, error)
	DeleteCharacterLooks(ctx context.Context, characterID uint32) error
}

func (q *queriesImpl) GetCharacterLooksByID(ctx context.Context, characterID uint32) (CharacterLooks, error) {
	var characterLooks CharacterLooks

	err := q.db.NewSelect().Model(&characterLooks).Where("character_id = ?", characterID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CharacterLooks{}, ErrNotFound
		}

		return CharacterLooks{}, err
	}

	return characterLooks, nil
}

func (q *queriesImpl) CreateCharacterLooks(ctx context.Context, characterLooks *CharacterLooks) (CharacterLooks, error) {
	_, err := q.db.NewInsert().Model(characterLooks).Exec(ctx)
	if err != nil {
		if isViolationOfConstraint(err, ConstraintCharacterLooksCharacterIDUnique) {
			return CharacterLooks{}, ErrCharacterIDNotUnique
		}

		return CharacterLooks{}, err
	}

	return *characterLooks, nil
}

func (q *queriesImpl) UpdateCharacterLooks(ctx context.Context, characterLooks *CharacterLooks) (CharacterLooks, error) {
	_, err := q.db.NewUpdate().Model(characterLooks).WherePK().Exec(ctx)
	if err != nil {
		return CharacterLooks{}, err
	}

	return *characterLooks, nil
}

func (q *queriesImpl) DeleteCharacterLooks(ctx context.Context, characterID uint32) error {
	_, err := q.db.NewDelete().Model((*CharacterLooks)(nil)).Where("character_id = ?", characterID).Exec(ctx)
	return err
}
