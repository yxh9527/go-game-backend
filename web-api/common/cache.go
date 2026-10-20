package common

//缓存中的玩家对

//链接访问策略
type UrlPloy struct {
	Ploys map[string]int
}

//代理相关
type ACCOUNT_TYPE int

const (
	M ACCOUNT_TYPE = 1 //总控账号
	A ACCOUNT_TYPE = 2 //信息账号
	S ACCOUNT_TYPE = 3 //代理账号
)

//地址策略管理对象
var urlPloys *UrlPloy

func NewUrlPloy() {
	if urlPloys == nil {
		u := &UrlPloy{
			Ploys: make(map[string]int),
		}
		urlPloys = u
	}
}

func UrlPloys() *UrlPloy {
	return urlPloys
}

func (u *UrlPloy) SetPloy(url string, accType []ACCOUNT_TYPE) {
	if accType == nil {
		return
	}
	urlCode := 0
	for _, t := range accType {
		urlCode = urlCode | (1 << t)
	}
	u.Ploys[url] = urlCode
}

func (u *UrlPloy) CheckPloy(url string, accType int) bool {
	code, ok := u.Ploys[url]
	if ok {
		return code&(1<<accType) > 0
	}
	return false
}
