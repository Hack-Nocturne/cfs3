package worker

import "gorm.io/gorm/clause"

func BulkAddObjects(objects []Object) error {
	err := db.Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(objects, 50).Error

	return err
}

func BulkRemoveObjects(ids []int64) error {
	err := db.Clauses(clause.OnConflict{DoNothing: true}).
		Delete(&Object{}, "id IN ?", ids).Error

	return err
}