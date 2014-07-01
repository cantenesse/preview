package storage

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/ngerakines/preview/common"
	"log"
	"strings"
	"time"
)

type MysqlManager struct {
	host, user, password, database string
}

func NewMysqlManager(host, user, password, database string) *MysqlManager {
	return &MysqlManager{host, user, password, database}
}

func (manager *MysqlManager) db() *sql.DB {
	url := fmt.Sprintf("%s:%s@tcp(%s)/%s", manager.user, manager.password, manager.host, manager.database)
	db, _ := sql.Open("mysql", url)
	return db
}

type mysqlSourceAssetStorageManager struct {
	manager *MysqlManager
	nodeId  string
}

type mysqlGeneratedAssetStorageManager struct {
	manager         *MysqlManager
	templateManager common.TemplateManager
	nodeId          string
}

func NewMysqlSourceAssetStorageManager(manager *MysqlManager, nodeId string) (common.SourceAssetStorageManager, error) {
	sasm := new(mysqlSourceAssetStorageManager)
	sasm.manager = manager
	sasm.nodeId = nodeId
	return sasm, nil
}

func NewMysqlGeneratedAssetStorageManager(manager *MysqlManager, templateManager common.TemplateManager, nodeId string) (common.GeneratedAssetStorageManager, error) {
	gasm := new(mysqlGeneratedAssetStorageManager)
	gasm.manager = manager
	gasm.templateManager = templateManager
	gasm.nodeId = nodeId
	return gasm, nil
}

func (sasm *mysqlSourceAssetStorageManager) Store(sourceAsset *common.SourceAsset) error {
	log.Println("About to store sourceAsset", sourceAsset)
	sourceAsset.CreatedBy = sasm.nodeId
	sourceAsset.UpdatedBy = sasm.nodeId
	payload, err := sourceAsset.Serialize()
	if err != nil {
		log.Println("Error serializing source asset:", err)
		return err
	}
	db := sasm.manager.db()

	statement, err := db.Prepare("INSERT INTO source_assets (id, type, message) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = statement.Exec(sourceAsset.Id, sourceAsset.IdType, payload)
	if err != nil {
		return err
	}

	return nil
}

func (sasm *mysqlSourceAssetStorageManager) FindBySourceAssetId(id string) ([]*common.SourceAsset, error) {
	db := sasm.manager.db()

	rows, err := db.Query("SELECT message FROM source_assets WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	results := make([]*common.SourceAsset, 0, 0)

	for rows.Next() {
		var message []byte
		err := rows.Scan(&message)
		if err == nil {
			sourceAsset, err := common.NewSourceAssetFromJson(message)
			if err != nil {
				return nil, err
			}
			results = append(results, sourceAsset)
		}
	}
	return results, nil
}

func (gasm *mysqlGeneratedAssetStorageManager) Store(generatedAsset *common.GeneratedAsset) error {
	log.Println("About to store generatedAsset", generatedAsset)
	generatedAsset.CreatedBy = gasm.nodeId
	generatedAsset.UpdatedBy = gasm.nodeId
	payload, err := generatedAsset.Serialize()
	if err != nil {
		log.Println("Error serializing source asset:", err)
		return err
	}

	log.Println("Storing generated asset", generatedAsset)

	db := gasm.manager.db()

	transaction, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = transaction.Exec(`INSERT INTO generated_assets (id, source, status, template_id, message) VALUES (?, ?, ?, ?, ?)`, generatedAsset.Id, generatedAsset.SourceAssetId, generatedAsset.Status, generatedAsset.TemplateId, payload)
	if err != nil {
		log.Println("Could not insert into generated_assets", err)
		defer transaction.Rollback()
		return err
	}

	if generatedAsset.Status == common.GeneratedAssetStatusWaiting {
		templateGroup, err := gasm.templateGroup(generatedAsset.TemplateId)
		if err != nil {
			log.Println("error getting template group", templateGroup)
			return err
		}
		_, err = transaction.Exec(`INSERT INTO waiting_generated_assets (id, source, template) VALUES (?, ?, ?)`, generatedAsset.Id, generatedAsset.SourceAssetId+generatedAsset.SourceAssetType, templateGroup)
		if err != nil {
			log.Println("Could not insert into waiting_generated_assets", err)
			defer transaction.Rollback()
			return err
		}
	}

	err = transaction.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (gasm *mysqlGeneratedAssetStorageManager) templateGroup(id string) (string, error) {
	templates, err := gasm.templateManager.FindByIds([]string{id})
	if err != nil {
		return "", err
	}
	if len(templates) != 1 {
		return "", common.ErrorNoTemplateForId
	}
	template := templates[0]
	return template.Group, nil
}

func (gasm *mysqlGeneratedAssetStorageManager) Update(generatedAsset *common.GeneratedAsset) error {
	generatedAsset.UpdatedAt = time.Now().UnixNano()
	generatedAsset.UpdatedBy = gasm.nodeId
	payload, err := generatedAsset.Serialize()
	if err != nil {
		log.Println("Error serializing generated asset:", err)
		return err
	}

	db := gasm.manager.db()

	transaction, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = transaction.Exec(`UPDATE generated_assets SET status = ?, message = ? WHERE id = ?`, generatedAsset.Status, payload, generatedAsset.Id)
	if err != nil {
		log.Println("Could not update generated_assets", err)
		defer transaction.Rollback()
		return err
	}

	if generatedAsset.Status == common.GeneratedAssetStatusScheduled || generatedAsset.Status == common.GeneratedAssetStatusProcessing {
		templateGroup, err := gasm.templateGroup(generatedAsset.TemplateId)
		if err != nil {
			return err
		}
		_, err = transaction.Exec(`DELETE FROM waiting_generated_assets WHERE id = ? AND template = ? AND source = ?`, generatedAsset.Id, templateGroup, generatedAsset.SourceAssetId+generatedAsset.SourceAssetType)
		if err != nil {
			log.Println("Could not delete from waiting_generated_assets", err)
			defer transaction.Rollback()
			return err
		}
		_, err = transaction.Exec(`INSERT INTO active_generated_assets (id) VALUES (?)`, generatedAsset.Id)
		if err != nil {
			log.Println("Could not insert into active_generated_assets", err)
			defer transaction.Rollback()
			return err
		}
	}
	if generatedAsset.Status == common.GeneratedAssetStatusComplete || strings.HasPrefix(generatedAsset.Status, common.GeneratedAssetStatusFailed) {
		_, err = transaction.Exec(`DELETE FROM active_generated_assets WHERE id = ?`, generatedAsset.Id)
		if err != nil {
			log.Println("Could not delete from waiting_generated_assets", err)
			defer transaction.Rollback()
			return err
		}
	}

	err = transaction.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (gasm *mysqlGeneratedAssetStorageManager) FindById(id string) (*common.GeneratedAsset, error) {
	generatedAssets, err := gasm.getIds([]string{id})
	if err != nil {
		return nil, err
	}
	if len(generatedAssets) == 0 {
		return nil, common.ErrorNoGeneratedAssetsFoundForId
	}
	return generatedAssets[0], nil
}

func (gasm *mysqlGeneratedAssetStorageManager) FindByIds(ids []string) ([]*common.GeneratedAsset, error) {
	return gasm.getIds(ids)
}

func (gasm *mysqlGeneratedAssetStorageManager) FindBySourceAssetId(id string) ([]*common.GeneratedAsset, error) {
	db := gasm.manager.db()

	rows, err := db.Query("SELECT message FROM generated_assets WHERE source = ?", id)
	if err != nil {
		return nil, err
	}

	return gasm.parseGeneratedAssetResults(rows)
}

func (gasm *mysqlGeneratedAssetStorageManager) FindWorkForService(serviceName string, workCount int) ([]*common.GeneratedAsset, error) {
	templates, err := gasm.templateManager.FindByRenderService(serviceName)
	if err != nil {
		log.Println("error executing templateManager.FindByRenderService", err)
		return nil, err
	}
	generatedAssetIds, err := gasm.getWaitingAssets(templates[0].Group, workCount)
	if err != nil {
		log.Println("error executing gasm.getWaitingAssets", err)
		return nil, err
	}

	return gasm.getIds(generatedAssetIds)
}

func (gasm *mysqlGeneratedAssetStorageManager) getWaitingAssets(group string, count int) ([]string, error) {
	db := gasm.manager.db()

	rows, err := db.Query(`SELECT id FROM waiting_generated_assets WHERE template = ? LIMIT ?`, group, count)
	if err != nil {
		return nil, err
	}

	results := make([]string, 0, 0)
	for rows.Next() {
		var generatedAssetId string
		err := rows.Scan(&generatedAssetId)
		if err == nil {
			results = append(results, generatedAssetId)
		}
	}
	return results, nil
}

func (gasm *mysqlGeneratedAssetStorageManager) getIds(ids []string) ([]*common.GeneratedAsset, error) {
	if len(ids) == 0 {
		return make([]*common.GeneratedAsset, 0), nil
	}

	args := make([]interface{}, len(ids))
	for i, v := range ids {
		args[i] = interface{}(v)
	}

	db := gasm.manager.db()

	rows, err := db.Query(`SELECT message FROM generated_assets WHERE id in (`+common.BuildIn(len(ids))+`)`, args...)
	if err != nil {
		return nil, err
	}

	return gasm.parseGeneratedAssetResults(rows)
}

func (gasm *mysqlGeneratedAssetStorageManager) parseGeneratedAssetResults(rows *sql.Rows) ([]*common.GeneratedAsset, error) {
	results := make([]*common.GeneratedAsset, 0, 0)
	for rows.Next() {
		var message []byte
		err := rows.Scan(&message)
		if err == nil {
			generatedAsset, err := common.NewGeneratedAssetFromJson(message)
			if err != nil {
				return nil, err
			}
			results = append(results, generatedAsset)
		}
	}
	return results, nil
}
