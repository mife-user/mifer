package loader

import (
	"context"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino-ext/components/document/loader/file"
)

// NewFileLoader 基于 Eino 官方 FileLoader 创建文档加载器，UseNameAsID 启用
func NewFileLoader(ctx context.Context) (document.Loader, error) {
	return file.NewFileLoader(ctx, &file.FileLoaderConfig{
		UseNameAsID: true,
	})
}
