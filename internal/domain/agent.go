package domain

type TalkReq struct {
	Content string
}

type MemoryReq struct {
	ID uint
}

type MemoryResp struct {
	Memory string
}
