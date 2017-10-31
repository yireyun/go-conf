package conf

import (
	"errors"
	"reflect"
	"sync"

	"github.com/Unknwon/goconfig"
)

type Config struct {
	curConfig *goconfig.ConfigFile //当前配置
	bakConfig *goconfig.ConfigFile //备份副本
	mu        sync.RWMutex
}

func LoadFile(fileName string) (s *Config, err error) {
	s = new(Config)
	s.curConfig, err = goconfig.LoadConfigFile(fileName)
	if err != nil {
		return nil, err
	}
	s.bakConfig, err = goconfig.LoadConfigFile(fileName)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Config) GetConfig(args interface{}) (err error) {
	if reflect.ValueOf(args).Kind() != reflect.Ptr || args == nil {
		return errors.New("参数不是指针或者为空")
	}

	v := reflect.ValueOf(args).Elem()
	t := v.Type()

	s.mu.RLock()
	defer s.mu.RUnlock()

	var str string
	for i := 0; i < v.NumField(); i++ {
		str, err = s.curConfig.GetValue(t.Name(), t.Field(i).Name)
		if err != nil {
			return err
		}
		v.Field(i).SetString(str)
	}
	return nil
}

func (s *Config) SetConfig(args interface{}) error {
	if reflect.ValueOf(args).Kind() != reflect.Ptr || args == nil {
		return errors.New("参数不是指针或者为空")
	}
	v := reflect.ValueOf(args).Elem()
	t := v.Type()

	s.mu.Lock()
	defer s.mu.Unlock()

	for i := 0; i < v.NumField(); i++ {
		s.curConfig.SetValue(t.Name(), t.Field(i).Name, v.Field(i).String())
	}

	return nil
}

func (s *Config) SaveFile(fileName string, bak bool) (err error) {
	var ok bool
	err, ok = s.configChange()
	if err != nil || ok {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if bak {
		err = goconfig.SaveConfigFile(s.bakConfig, fileName+".bak")
		if err != nil {
			return err
		}
	}

	err = goconfig.SaveConfigFile(s.curConfig, fileName)
	if err != nil {
		return err
	}
	s.bakConfig, err = goconfig.LoadConfigFile(fileName)
	if err != nil {
		return err
	}
	return
}

func (s *Config) configChange() (error, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	curSections := s.curConfig.GetSectionList()
	bakSections := s.bakConfig.GetSectionList()
	if len(curSections) != len(bakSections) {
		return nil, false
	}
	//判断curSections是否存在
forCur:
	for _, curSection := range curSections {
		for _, bakSection := range bakSections {
			if bakSection == curSection {
				continue forCur
			}
		}
		return nil, false
	}
forBak:
	//判断bakSections是否存在
	for _, bakSection := range bakSections {
		for _, curSection := range curSections {
			if bakSection == curSection {
				continue forBak
			}
		}
		return nil, false
	}

	for _, curSection := range curSections {
		curMap, err := s.curConfig.GetSection(curSection)
		if err != nil {
			return err, false

		}
		bakMap, err := s.bakConfig.GetSection(curSection)
		if err != nil {
			return err, false
		}
		//判断curSections中的值是否存在并且相等
		for curKey, curValue := range curMap {
			if bakValue, ok := bakMap[curKey]; !ok || bakValue != curValue {
				return nil, false
			}
		}
		//判断bakSections中的值是否存在并且相等
		for bakKey, bakValue := range bakMap {
			if curValue, ok := curMap[bakKey]; !ok || bakValue != curValue {
				return nil, false
			}
		}
	}
	return nil, true
}

func (s *Config) ReLoad() (err error) {
	err = s.curConfig.Reload()
	if err != nil {
		return err
	}
	return s.bakConfig.Reload()
}
