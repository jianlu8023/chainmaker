package entity

const factTableName = "fact"

type FactEntity struct {
	Id        float64 `json:"id" gorm:"primaryKey,not null;autoIncrement;column:id;type:bigint;comment:id"`
	FileHash  string  `json:"file_hash" gorm:"column:file_hash;type:varchar(255);index:idx_file_hash;comment:文件hash"`
	FileName  string  `json:"file_name" gorm:"column:file_name;type:varchar(255);comment:文件名"`
	TimeStamp int64   `json:"time_stamp" gorm:"column:time_stamp;type:bigint;comment:时间戳"`
	TxId      string  `json:"tx_id" gorm:"column:tx_id;type:varchar(255);index:idx_tx_id;comment:交易id"`
}

func (FactEntity) TableName() string {
	return factTableName
}
