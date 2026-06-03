package service

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"sync"

	"github.com/getcharzp/go-ocr/ddddocr"
)

var ddddOcrCache = struct {
	sync.Mutex
	key    string
	engine *ddddocr.Engine
}{}

func SolveDdddOcrCaptcha(conf Config, imageBytes []byte) (string, error) {
	if conf.DdddOcrOnnxRuntimeLibPath == "" || conf.DdddOcrModelPath == "" {
		return "", errors.New("ddddocr requires DdddOcrOnnxRuntimeLibPath and DdddOcrModelPath")
	}
	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return "", err
	}
	engine, err := ddddOcrEngine(conf)
	if err != nil {
		return "", err
	}
	return engine.Classification(img)
}

func ddddOcrEngine(conf Config) (*ddddocr.Engine, error) {
	key := conf.DdddOcrOnnxRuntimeLibPath + "|" + conf.DdddOcrModelPath + "|" + conf.DdddOcrDetModelPath
	ddddOcrCache.Lock()
	defer ddddOcrCache.Unlock()
	if ddddOcrCache.engine != nil && ddddOcrCache.key == key {
		return ddddOcrCache.engine, nil
	}
	engine, err := ddddocr.NewEngine(ddddocr.Config{
		OnnxRuntimeLibPath: conf.DdddOcrOnnxRuntimeLibPath,
		ModelPath:          conf.DdddOcrModelPath,
		DetModelPath:       conf.DdddOcrDetModelPath,
		UseCustomModel:     conf.DdddOcrUseCustomModel,
	})
	if err != nil {
		return nil, err
	}
	ddddOcrCache.key = key
	ddddOcrCache.engine = engine
	return engine, nil
}
