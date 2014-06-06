package common

import (
	"github.com/ziutek/mymysql/autorc"
	"github.com/ziutek/mymysql/mysql"
	_ "github.com/ziutek/mymysql/thrsafe" // You may also use the native engine
	"log"
	"strings"
	"time"
)

type mysqlManager struct {
	connection *autorc.Conn
}

func newMysqlManager(host, user, password, database string) *mysqlManager {
	conn := autorc.New("tcp", "", host, user, password, database)
	return &mysqlManager{conn}
}

func (manager *mysqlManager) db() *autorc.Conn {
	return manager.connection
}

type mysqlSourceAssetStorageManager struct {
	manager *mysqlManager
	nodeId  string
}

type mysqlGeneratedAssetStorageManager struct {
	manager         *mysqlManager
	templateManager TemplateManager
	nodeId          string
}

func NewMysqlSourceAssetStorageManager(manager *mysqlManager, nodeId string) (SourceAssetStorageManager, error) {
	sasm := new(mysqlSourceAssetStorageManager)
	sasm.manager = manager
	sasm.nodeId = nodeId
	return sasm, nil
}

func NewMysqlGeneratedAssetStorageManager(manager *mysqlManager, templateManager TemplateManager, nodeId string) (GeneratedAssetStorageManager, error) {
	gasm := new(mysqlGeneratedAssetStorageManager)
	gasm.manager = manager
	gasm.templateManager = templateManager
	gasm.nodeId = nodeId
	return gasm, nil
}

func (sasm *mysqlSourceAssetStorageManager) Store(sourceAsset *SourceAsset) error {
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
	_, _, err = statement.Exec(sourceAsset.Id, sourceAsset.IdType, payload)
	if err != nil {
		return err
	}

	return nil
}

func (sasm *mysqlSourceAssetStorageManager) FindBySourceAssetId(id string) ([]*SourceAsset, error) {
	db := sasm.manager.db()

	rows, _, err := db.Query("SELECT message FROM source_assets WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	results := make([]*SourceAsset, 0, 0)

	for _, row := range rows {
		message := row[0].([]byte)
		sourceAsset, err := newSourceAssetFromJson(message)
		if err != nil {
			return nil, err
		}
		results = append(results, sourceAsset)
	}
	return results, nil
}

func (gasm *mysqlGeneratedAssetStorageManager) Store(generatedAsset *GeneratedAsset) error {
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

	err = db.Begin(func(tr mysql.Transaction, args ...interface{}) error {
		// This function will be called again if returns a recoverable error
		generatedAssetInsert, err := db.Prepare(`INSERT INTO generated_assets (id, source, status, template_id, message) VALUES (?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		generatedAssetInsert.Bind(generatedAssetInsert, generatedAsset.Id, generatedAsset.SourceAssetId, generatedAsset.Status, generatedAsset.TemplateId, payload)
		s1 := tr.Do(generatedAssetInsert.Raw)
		if _, err := s1.Run(); err != nil {
			return err
		}

		if generatedAsset.Status == GeneratedAssetStatusWaiting {
			log.Println("generated asset status is", GeneratedAssetStatusWaiting)
			templateGroup, err := gasm.templateGroup(generatedAsset.TemplateId)
			if err != nil {
				log.Println("error getting template group", templateGroup)
				return err
			}
			waitingGeneratedAssetInsert, err := db.Prepare(`INSERT INTO waiting_generated_assets (id, source, template) VALUES (?, ?, ?)`)
			waitingGeneratedAssetInsert.Bind(generatedAsset.Id, generatedAsset.SourceAssetId+generatedAsset.SourceAssetType, templateGroup)
			s2 := tr.Do(waitingGeneratedAssetInsert.Raw)
			if _, err := s2.Run(); err != nil {
				return err
			}
		}

		return tr.Commit()
	})

	if err != nil {
		log.Println("Error executing batch:", err)
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
		return "", ErrorNoTemplateForId
	}
	template := templates[0]
	return template.Group, nil
}

func (gasm *mysqlGeneratedAssetStorageManager) Update(generatedAsset *GeneratedAsset) error {
	generatedAsset.UpdatedAt = time.Now().UnixNano()
	generatedAsset.UpdatedBy = gasm.nodeId
	payload, err := generatedAsset.Serialize()
	if err != nil {
		log.Println("Error serializing generated asset:", err)
		return err
	}

	db := gasm.manager.db()

	err = db.Begin(func(tr mysql.Transaction, args ...interface{}) error {
		// This function will be called again if returns a recoverable error
		generatedAssetUpdate, err := db.Prepare(`UPDATE generated_assets SET status = ?, message = ? WHERE id = ?`)
		if err != nil {
			return err
		}
		generatedAssetUpdate.Bind(generatedAsset.Status, payload, generatedAsset.Id)
		s1 := tr.Do(generatedAssetUpdate.Raw)
		if _, err := s1.Run(); err != nil {
			return err
		}

		if generatedAsset.Status == GeneratedAssetStatusScheduled || generatedAsset.Status == GeneratedAssetStatusProcessing {
			templateGroup, err := gasm.templateGroup(generatedAsset.TemplateId)
			if err != nil {
				return err
			}
			s3i, err := db.Prepare(`DELETE FROM waiting_generated_assets WHERE id = ? AND template = ? AND source = ?`)
			if err != nil {
				return err
			}
			s3i.Bind(generatedAsset.Id, templateGroup, generatedAsset.SourceAssetId+generatedAsset.SourceAssetType)
			s3 := tr.Do(s3i.Raw)
			if _, err := s3.Run(); err != nil {
				return err
			}
			s4i, err := db.Prepare(`INSERT INTO active_generated_assets (id) VALUES (?)`)
			s4i.Bind(generatedAsset.Id)
			s4 := tr.Do(s4i.Raw)
			if _, err := s4.Run(); err != nil {
				return err
			}
		}
		if generatedAsset.Status == GeneratedAssetStatusComplete || strings.HasPrefix(generatedAsset.Status, GeneratedAssetStatusFailed) {
			s3i, err := db.Prepare(`DELETE FROM active_generated_assets WHERE id = ?`)
			if err != nil {
				return err
			}
			s3i.Bind(generatedAsset.Id)
			s3 := tr.Do(s3i.Raw)
			if _, err := s3.Run(); err != nil {
				return err
			}
		}

		return tr.Commit()
	})

	if err != nil {
		log.Println("Error executing batch:", err)
		return err
	}
	return nil
}

func (gasm *mysqlGeneratedAssetStorageManager) FindById(id string) (*GeneratedAsset, error) {
	generatedAssets, err := gasm.getIds([]string{id})
	if err != nil {
		return nil, err
	}
	if len(generatedAssets) == 0 {
		return nil, ErrorNoGeneratedAssetsFoundForId
	}
	return generatedAssets[0], nil
}

func (gasm *mysqlGeneratedAssetStorageManager) FindByIds(ids []string) ([]*GeneratedAsset, error) {
	return gasm.getIds(ids)
}

func (gasm *mysqlGeneratedAssetStorageManager) FindBySourceAssetId(id string) ([]*GeneratedAsset, error) {
	results := make([]*GeneratedAsset, 0, 0)

	db := gasm.manager.db()

	rows, _, err := db.Query("SELECT message FROM generated_assets WHERE source = ?", id)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		message := row[0].([]byte)
		generatedAsset, err := newGeneratedAssetFromJson(message)
		if err != nil {
			return nil, err
		}
		results = append(results, generatedAsset)
	}
	return results, nil
}

func (gasm *mysqlGeneratedAssetStorageManager) FindWorkForService(serviceName string, workCount int) ([]*GeneratedAsset, error) {
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
	results := make([]string, 0, 0)

	db := gasm.manager.db()

	rows, _, err := db.Query("SELECT id FROM waiting_generated_assets WHERE template = ? LIMIT ?", group, count)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		generatedAssetId := row[0].(string)
		results = append(results, generatedAssetId)
	}
	return results, nil
}

func (gasm *mysqlGeneratedAssetStorageManager) getIds(ids []string) ([]*GeneratedAsset, error) {
	results := make([]*GeneratedAsset, 0, 0)

	args := make([]interface{}, len(ids))
	for i, v := range ids {
		args[i] = interface{}(v)
	}

	db := gasm.manager.db()

	rows, _, err := db.Query(`SELECT message FROM generated_assets WHERE id in (`+buildIn(len(ids))+`)`, args...)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		message := row[0].([]byte)
		generatedAsset, err := newGeneratedAssetFromJson(message)
		if err != nil {
			return nil, err
		}
		results = append(results, generatedAsset)
	}
	return results, nil
}
