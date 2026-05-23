package chunker

import "github.com/cloudwego/eino/components/document"

// dedupSplitter 在递归分块后对内容做 SHA256 去重，并为每条分块生成唯一 ID
type dedupSplitter struct {
	splitter document.Transformer
}
