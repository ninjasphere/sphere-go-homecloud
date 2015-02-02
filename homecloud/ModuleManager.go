package homecloud

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/go-ninja/rpc/json2"
	"github.com/ninjasphere/sphere-go-homecloud/models"
)

type ModuleManager struct {
	Conn        *ninja.Connection   `inject:""`
	ModuleModel *models.ModuleModel `inject:""`
	log         *logger.Logger
}

func (m *ModuleManager) PostConstruct() error {
	m.log = logger.GetLogger("ModuleManager")
	return m.Start()
}

func (m *ModuleManager) Start() error {
	m.Conn.Subscribe("$node/:node/:type/:module/event/announce", func(announcement *json.RawMessage, values map[string]string) bool {

		node, moduleType, moduleName := values["node"], values["type"], values["module"]

		log.Infof("Got app announcement node:%s module:%s type:%s", node, moduleName, moduleType)

		if announcement == nil {
			log.Warningf("Nil app announcement from node:%s module:%s type:%s", node, moduleName, moduleType)
			return true
		}

		module := &model.Module{}
		err := json.Unmarshal(*announcement, module)

		if announcement == nil {
			log.Warningf("Could not parse announcement from node:%s module:%s  type:%s error:%s", node, moduleName, err)
			return true
		}

		err = m.ModuleModel.Create(module)
		if err != nil {
			log.Warningf("Failed to save module announcement for %s error:%s", moduleName, err)
		}

		config, err := m.ModuleModel.GetConfig(values["module"])

		if err != nil {
			log.Warningf("Failed to retrieve config for module %s error:%s", moduleName, err)
		} else {
			err = m.startModule(fmt.Sprintf("$node/%s/%s/%s", node, moduleType, moduleName), module, config)
			if err != nil {
				log.Warningf("Failed to start module:%s on node:%s error:%s", moduleName, err)
			}
		}

		return true
	})

	return m.Conn.Subscribe("$node/:node/:type/:module/event/config", func(config *json.RawMessage, values map[string]string) bool {
		log.Infof("Got module config node:%s module:%s config:%s", values["node"], values["module"], *config)

		if config != nil {
			err := m.ModuleModel.SetConfig(values["module"], string(*config))

			if err != nil {
				log.Warningf("Failed to save config for module: %s error: %s", values["module"], err)
			}
		} else {
			log.Infof("Nil config recevied from node:%s module:%s", values["node"], values["module"])
		}

		return true
	})

}

func (m *ModuleManager) startModule(topic string, module *model.Module, config *string) error {

	var rawConfig json.RawMessage
	if config != nil {
		rawConfig = []byte(*config)
	} else {
		rawConfig = []byte("{}")
	}

	client := m.Conn.GetServiceClient(topic)
	err := client.Call("start", &rawConfig, nil, 10*time.Second)

	if err != nil {
		jsonError, ok := err.(*json2.Error)
		if ok {
			if jsonError.Code == json2.E_INVALID_REQ {

				err := m.ModuleModel.DeleteConfig(module.ID)
				if err != nil {
					log.Warningf("Module %s could not parse its config. Also, we couldn't clear it! errors:%s and %s", module.ID, jsonError.Message, err)
				} else {
					log.Warningf("Module %s could not parse its config, so we cleared it from redis. error:%s", module.ID, jsonError.Message)
				}

				return m.startModule(topic, module, nil)
			}
		}
	}

	return err

}
