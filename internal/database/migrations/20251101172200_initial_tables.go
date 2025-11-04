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
			Model((*Account20251101172200)(nil)).
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

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		_, err := db.NewDropTable().
			Model((*AccountSession20251101172200)(nil)).
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

		return nil
	})
}

type Account20251101172200 struct {
	bun.BaseModel `bun:"table:accounts"`

	ID       uint32 `bun:"id,pk,autoincrement,type:int unsigned"`
	Username string `bun:"type:varchar(16),notnull,unique"`
	Password string `bun:"type:varchar(64),notnull"`

	CreatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
}

type AccountSession20251101172200 struct {
	bun.BaseModel `bun:"table:account_sessions"`

	AccountID     uint32 `bun:"type:int unsigned,unique"`
	CharacterID   uint32 `bun:"type:int unsigned,notnull,pk"`
	SessionKey    string `bun:"type:varchar(16),notnull,unique"`
	ClientAddress uint32 `bun:"type:int unsigned,notnull"`

	CreatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
}
