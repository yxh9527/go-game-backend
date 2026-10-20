package common

import "strconv"

//接口类型
type API_TYPE int

const (
	API_LOGIN               API_TYPE = 0
	API_BALANCE             API_TYPE = 1
	API_ADD_SCORE           API_TYPE = 2
	API_SUB_SCORE           API_TYPE = 3
	API_GAME_DATA           API_TYPE = 6
	API_WEB_AUTH            API_TYPE = 7
	API_KICK_OUT            API_TYPE = 9
	API_UPDATE_CURRENCYTYPE API_TYPE = 10
)

func (a API_TYPE) String() string {
	return strconv.Itoa(int(a))
}

type Status int

const (
	//状态码
	CODE_OK                           Status = 0
	CODE_TOKEN_LOSS                   Status = 1
	CODE_AGENT_NOT                    Status = 2
	CODE_CHECK_TIMEOUT                Status = 3
	CODE_CHECK_ERR                    Status = 4
	CODE_AGENT_WHITE_ERR              Status = 5
	CODE_CHECK_FIELD_LOSS             Status = 6
	CODE_TOKEN_INVALID                Status = 7
	CODE_REQUEST_NOT                  Status = 8
	CODE_IN_GAME_REPEAT               Status = 10
	CODE_ACCOUNT_NOT                  Status = 11
	CODE_ACCOUNT_NLT_AGENT            Status = 12
	CODE_ACCOUNT_NOT_PROXY            Status = 13
	CODE_ACCOUNT_IN_GAME              Status = 14
	CODE_AGENT_CHECK_ERR              Status = 15
	CODE_DATA_NOT                     Status = 16
	CODE_GAME_RETURN_ERR              Status = 17
	CODE_TAKEOUT_MONEY_OID_NOT_PROXY  Status = 18
	CODE_CONFIG_MONEY_OID_NOT_PROXY   Status = 19
	CODE_ACCOUNT_PROHIBIT             Status = 20
	CODE_ACCOUNT_SIGN_OUT             Status = 21
	CODE_AES_DECRYPT_ERR              Status = 22
	CODE_URL_DECRYPT_ERR              Status = 23
	CODE_AGENT_GET_DATA_TIMEOUT       Status = 24
	CODE_GAME_NOT                     Status = 25
	CODE_ORDER_NOT                    Status = 26
	CODE_DATABASE_ABNORMAL            Status = 27
	CODE_IP_PROHIBIT                  Status = 28
	CODE_ORDERID_FORMAT_ERR           Status = 29
	CODE_ACCOUNT_OLSTATE_ERR          Status = 30
	CODE_SCORE_LESS_EQUAL             Status = 31
	CODE_UP_ACCOUNT_INFO_ERR          Status = 32
	CODE_UP_ACCOUNT_SCORE_ERR         Status = 33
	CODE_ORDER_REPEAT                 Status = 34
	CODE_GET_ACCOUNT_INFO_ERR         Status = 35
	CODE_KINDID_NOT                   Status = 36
	CODE_LOGIN_PROHIBIT_LOWER_SCORE   Status = 37
	CODE_ACCOUNT_BALANCE_INSUFFICIENT Status = 38
	CODE_DUPLITCATE_LOGON             Status = 39
	CODE_LOWER_SCORE_EXCEEDING_LIMIT  Status = 40
	CODE_GET_COUNT_TIME_ERR           Status = 41
	CODE_AGENT_PROHIBIT               Status = 42
	CODE_GET_ORDER_FREQUENTLY         Status = 43
	CODE_ORDER_IN_PROCESSING          Status = 44
	CODE_GAME_PROHIBIT                Status = 45
	CODE_WEB_PROHIBIT                 Status = 46
	CODE_PARAM_CHECK_FAIL             Status = 47
	CODE_FIND_SCORE                   Status = 49
	CODE_TAKE_ERR                     Status = 50
	CODE_REQUEST_ERR                  Status = 999
	CODE_ACCOUNT_ABNORMAL             Status = 1001
	CODE_AGENT_BALANCE_INSUFFICIENT   Status = 1002
)

var statusDesc = map[Status]string{
	0: "成功",
	1: "TOKEN丢失",
	2: "渠道不存在",
	3: "验证时间超时",
	4: "验证错误",
	5: "渠道白名单错误",
	6: "验证字段丢失",
	7: "TOKEN失效",
	8: "不存在的请求",

	10:   "玩家同时在多款游戏中",
	11:   "玩家帐号不存在",
	12:   "玩家帐号在渠道中不存",
	13:   "玩家登录帐号不匹配",
	14:   "玩家正在游戏中",
	15:   "渠道验证错误",
	16:   "数据不存在",
	17:   "游戏返回码不存在",
	18:   "取出游戏的钱和订单号不匹配",
	19:   "设置游戏的钱和订单号不匹配",
	20:   "账号禁用",
	21:   "账号踢出",
	22:   "SAES解密失败",
	23:   "URL解码失败",
	24:   "渠道拉取数据超过时间范围",
	25:   "游戏不存在",
	26:   "订单号不存在",
	27:   "数据库异常",
	28:   "IP禁用",
	29:   "订单号与规则不符",
	30:   "获取玩家在线状态失败",
	31:   "更新的分数小于或者等0",
	32:   "更新玩家信息失败",
	33:   "更新玩家金币失败",
	34:   "订单重复",
	35:   "获取玩家信息失败",
	36:   "KindID不存在",
	37:   "登录瞬间禁止下分，导致下分失败",
	38:   "余额不足导致下分失败",
	39:   "禁止同一账号登录带分、上分、下分并发请求，后一个请求被拒",
	40:   "单次上下分数量不能超过一千万",
	41:   "拉取对局汇总统计时间范围有误",
	42:   "代理被禁用",
	43:   "拉单过亍频繁(两次拉单时间间隔必须大于1秒)",
	44:   "订单正在处理中",
	47:   "参数错误",
	49:   "余额查询失败",
	50:   "踢人失败",
	999:  "请求失败",
	1001: "注册会员账号系统异常",
	1002: "代理商金额不足",
}

//获取描述
func (s Status) String() string {
	if str, ok := statusDesc[s]; ok {
		return str
	}
	return ""
}
