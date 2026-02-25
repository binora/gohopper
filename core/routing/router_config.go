package routing

type RouterConfig struct {
	MaxVisitedNodes int
}

func NewRouterConfig() RouterConfig {
	return RouterConfig{MaxVisitedNodes: int(^uint(0) >> 1)}
}
