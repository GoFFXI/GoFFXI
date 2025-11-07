// SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: LicenseRef-NvidiaProprietary
//
// NVIDIA CORPORATION, its affiliates and licensors retain all intellectual
// property and proprietary rights in and to this material, related
// documentation and any modifications thereto. Any use, reproduction,
// disclosure or distribution of this material and related documentation
// without an express license agreement from NVIDIA CORPORATION or
// its affiliates is strictly prohibited.

package migrations

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

//nolint:gochecknoinits // this is the typical way to register bun migrations
func init() {
	migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		_, err := db.NewCreateTable().
			Model((*AccountBan20251101172200)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*AccountIPRecord20251101172200)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*AccountSession20251101172200)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*AccountTOTP20251101172200)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*Account20251101172200)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*CharacterJobs20251101172200)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*CharacterLooks20251101172200)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*CharacterStats20251101172200)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().
			Model((*Character20251101172200)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		_, err := db.NewDropTable().
			Model((*AccountBan20251101172200)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().
			Model((*AccountIPRecord20251101172200)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().
			Model((*AccountSession20251101172200)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().
			Model((*AccountTOTP20251101172200)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().
			Model((*Account20251101172200)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().
			Model((*CharacterJobs20251101172200)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().
			Model((*CharacterLooks20251101172200)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().
			Model((*CharacterStats20251101172200)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().
			Model((*Character20251101172200)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

type AccountBan20251101172200 struct {
	bun.BaseModel `bun:"table:account_bans"`

	AccountID    uint32    `bun:"type:int unsigned,unique,pk"`
	TimeBanned   time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
	TimeUnbanned time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
	Reason       string    `bun:"type:varchar(512),notnull"`
}

type AccountIPRecord20251101172200 struct {
	bun.BaseModel `bun:"table:account_ip_records"`

	LoginTime   time.Time `bun:"type:timestamp,notnull,default:current_timestamp,pk"`
	AccountID   uint32    `bun:"type:int unsigned,notnull,pk"`
	CharacterID uint32    `bun:"type:int unsigned,notnull"`
	ClientIP    string    `bun:"client_ip,notnull"`
}

type AccountSession20251101172200 struct {
	bun.BaseModel `bun:"table:account_sessions"`

	AccountID   uint32 `bun:"type:int unsigned,unique"`
	CharacterID uint32 `bun:"type:int unsigned,notnull,pk"`
	SessionKey  string `bun:"type:varchar(16),notnull,unique"`
	ClientIP    string `bun:"type:varchar(15),notnull"`

	CreatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
}

type AccountTOTP20251101172200 struct {
	bun.BaseModel `bun:"table:account_totps"`

	AccountID    uint32 `bun:"type:int unsigned,notnull,pk"`
	Secret       string `bun:"type:varchar(32),notnull"`
	RecoveryCode string `bun:"type:varchar(32),notnull"`
	Validated    bool   `bun:"type:boolean,notnull,default:false"`
}

type Account20251101172200 struct {
	bun.BaseModel `bun:"table:accounts"`

	ID       uint32 `bun:"id,pk,autoincrement,type:int unsigned"`
	Username string `bun:"type:varchar(16),notnull,unique"`
	Password string `bun:"type:varchar(64),notnull"`

	CreatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
}

type CharacterJobs20251101172200 struct {
	bun.BaseModel `bun:"table:character_jobs"`

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

type CharacterLooks20251101172200 struct {
	bun.BaseModel `bun:"table:character_looks"`

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

type CharacterStats20251101172200 struct {
	bun.BaseModel `bun:"table:character_stats"`

	CharacterID uint32 `bun:"type:int unsigned,unique,pk"`
	HP          uint16 `bun:"type:smallint unsigned,notnull,default:50"`
	MP          uint16 `bun:"type:smallint unsigned,notnull,default:50"`
	MainJob     uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
	SubJob      uint8  `bun:"type:tinyint unsigned,notnull,default:0"`
}

type Character20251101172200 struct {
	bun.BaseModel `bun:"table:characters"`

	ID                uint32 `bun:"type:int unsigned,unique,pk,autoincrement"`
	AccountID         uint32 `bun:"type:int unsigned"`
	OriginalAccountID uint32 `bun:"type:int unsigned"`
	Name              string `bun:"type:varchar(16),notnull,unique"`
	Nation            uint8  `bun:"type:tinyint unsigned,notnull"`
	PosZone           uint16 `bun:"type:smallint unsigned,notnull"`
}
