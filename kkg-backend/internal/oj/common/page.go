package common

type PageRequest struct {
	Current   int64  `json:"current"`
	PageSize  int64  `json:"pageSize"`
	SortField string `json:"sortField"`
	SortOrder string `json:"sortOrder"`
}

func (p *PageRequest) Normalize() {
	if p.Current <= 0 {
		p.Current = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 10
	}
}

type PageResult struct {
	Records interface{} `json:"records"`
	Total   int64       `json:"total"`
	Size    int64       `json:"size"`
	Current int64       `json:"current"`
}

type DeleteRequest struct {
	ID int64 `json:"id"`
}
