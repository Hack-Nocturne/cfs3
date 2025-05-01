package worker

var GlobalObjects []Object

type Object struct {
	ID          int64  `gorm:"primaryKey"`
	Hash        string
	RelPath     string `gorm:"uniqueIndex"`
	Name        string
	AddedBy     *string
	ProjectName string `gorm:"index"`
	Metadata    *string
}
