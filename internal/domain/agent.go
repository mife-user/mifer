package domain

type TalkReq struct {
	content string
}

type TalkResp struct {
	content string
}

type MemoryReq struct {
	ID uint
}

type MemoryResp struct {
	Memory string
}
