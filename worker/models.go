package worker

import "github.com/Hack-Nocturne/cfs3/types"

type Object struct {
	ID          int64 `gorm:"primaryKey"`
	types.FileContainer
	Name        string
	AddedBy     *string
	ProjectName string `gorm:"index"`
	Metadata    *string
}
