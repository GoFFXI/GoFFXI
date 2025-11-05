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
	ID        uint32 `bun:"type:int unsigned,unique,pk,autoincrement"`
	AccountID uint32 `bun:"type:int unsigned"`
	Name      string `bun:"type:varchar(16),notnull,unique"`
	Nation    uint8  `bun:"type:tinyint unsigned,notnull"`
}

type CharacterQueries interface {
	GetCharacterByID(ctx context.Context, characterID uint32) (Character, error)
	CreateCharacter(ctx context.Context, character *Character) (Character, error)
	UpdateCharacter(ctx context.Context, character *Character) (Character, error)
	DeleteCharacter(ctx context.Context, characterID uint32) error
}

func (q *queriesImpl) GetCharacterByID(ctx context.Context, characterID uint32) (Character, error) {
	var character Character

	err := q.db.NewSelect().Model(&character).Where("character_id = ?", characterID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Character{}, ErrNotFound
		}

		return Character{}, err
	}

	return character, nil
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
